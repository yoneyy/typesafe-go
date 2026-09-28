package typesafe

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type closeTrackingBody struct {
	io.Reader
	closed bool
}

func (b *closeTrackingBody) Close() error {
	b.closed = true
	return nil
}

type failingBody struct {
	err    error
	closed bool
}

func (b *failingBody) Read([]byte) (int, error) { return 0, b.err }
func (b *failingBody) Close() error {
	b.closed = true
	return nil
}

func TestApiErrorMessage(t *testing.T) {
	tests := []struct {
		name string
		err  ApiError
		want string
	}{
		{"detail", ApiError{StatusCode: 422, Detail: json.RawMessage(`"invalid"`), Body: []byte(`ignored`)}, `typesafe: http 422: "invalid"`},
		{"body fallback", ApiError{StatusCode: 502, Body: []byte(`gateway error`)}, `typesafe: http 502: gateway error`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.want {
				t.Fatalf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTransportPreservesBaseURLPath(t *testing.T) {
	tests := []struct {
		name    string
		baseURL string
		path    string
		wantURL string
	}{
		{"version prefix", "https://api.typesafe.ai/v1", SystemOnePath, "https://api.typesafe.ai/v1/systemone"},
		{"version prefix with trailing slash", "https://api.typesafe.ai/v1/", SystemOnePath, "https://api.typesafe.ai/v1/systemone"},
		{"root URL", "https://example.test", ModelsPath, "https://example.test/models"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := clientWithTransport(func(req *http.Request) (*http.Response, error) {
				if got := req.URL.String(); got != tt.wantURL {
					t.Errorf("request URL = %q, want %q", got, tt.wantURL)
				}
				return testResponse(http.StatusOK, `{}`), nil
			})
			client.baseURL = tt.baseURL
			if _, err := client.doRaw(context.Background(), http.MethodPost, tt.path, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTransportNonRetryableErrors(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantDetail string
		wantError  string
	}{
		{"string detail", `{"detail":"invalid question"}`, `"invalid question"`, `typesafe: http 422: "invalid question"`},
		{"object detail", `{"detail":{"field":"criteria"}}`, `{"field":"criteria"}`, `typesafe: http 422: {"field":"criteria"}`},
		{"plain body", `bad request`, ``, `typesafe: http 422: bad request`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := clientWithTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return testResponse(http.StatusUnprocessableEntity, tt.body), nil
			})
			client.maxRetries = 2
			_, err := client.Models(context.Background())
			var apiErr ApiError
			if !errors.As(err, &apiErr) {
				t.Fatalf("error = %v, want ApiError", err)
			}
			if calls != 1 || apiErr.StatusCode != http.StatusUnprocessableEntity || string(apiErr.Body) != tt.body || string(apiErr.Detail) != tt.wantDetail {
				t.Fatalf("calls=%d, ApiError=%+v", calls, apiErr)
			}
			if got := apiErr.Error(); got != tt.wantError {
				t.Errorf("error message = %q, want %q", got, tt.wantError)
			}
		})
	}
}

func TestTransportRetriesTransientFailures(t *testing.T) {
	tests := []struct {
		name   string
		status int
	}{
		{"rate limit", http.StatusTooManyRequests},
		{"server error", http.StatusInternalServerError},
		{"overloaded", 529},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			client := clientWithTransport(func(*http.Request) (*http.Response, error) {
				calls++
				if calls == 1 {
					return testResponse(tt.status, `{"detail":"temporary"}`), nil
				}
				return testResponse(http.StatusOK, `{"models":[]}`), nil
			})
			client.maxRetries = 1
			models, err := client.Models(context.Background())
			if err != nil || models == nil || calls != 2 {
				t.Fatalf("Models() = (%v, %v), calls=%d; want success after one retry", models, err, calls)
			}
		})
	}
}

func TestTransportRetriesNetworkErrorAndReplaysBody(t *testing.T) {
	temporary := errors.New("temporary network failure")
	var bodies [][]byte
	client := clientWithTransport(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		bodies = append(bodies, body)
		if len(bodies) == 1 {
			return nil, temporary
		}
		return testResponse(http.StatusOK, `{"ok":true}`), nil
	})
	client.maxRetries = 1
	got, err := client.doRaw(context.Background(), http.MethodPost, "/test", map[string]string{"prompt": "hello"})
	if err != nil || string(got) != `{"ok":true}` || len(bodies) != 2 {
		t.Fatalf("doRaw() = (%s, %v), calls=%d", got, err, len(bodies))
	}
	for _, body := range bodies {
		assertJSONEqual(t, `{"prompt":"hello"}`, body)
	}
}

func TestTransportStopsAtRetryLimit(t *testing.T) {
	calls := 0
	client := clientWithTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return testResponse(http.StatusServiceUnavailable, `{"detail":"unavailable"}`), nil
	})
	client.maxRetries = 1
	_, err := client.Models(context.Background())
	var apiErr ApiError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusServiceUnavailable || calls != 2 {
		t.Fatalf("error = %v, calls=%d; want 503 after one retry", err, calls)
	}
}

func TestTransportClosesBodyBeforeRetry(t *testing.T) {
	firstBody := &closeTrackingBody{Reader: strings.NewReader(`{"detail":"temporary"}`)}
	calls := 0
	client := clientWithTransport(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return &http.Response{StatusCode: http.StatusInternalServerError, Header: make(http.Header), Body: firstBody}, nil
		}
		if !firstBody.closed {
			t.Error("first response body was still open before retry")
		}
		return testResponse(http.StatusOK, `{"models":[]}`), nil
	})
	client.maxRetries = 1
	if _, err := client.Models(context.Background()); err != nil {
		t.Fatal(err)
	}
	if calls != 2 || !firstBody.closed {
		t.Fatalf("calls=%d, first body closed=%t", calls, firstBody.closed)
	}
}

func TestTransportReadErrorClosesBody(t *testing.T) {
	readErr := errors.New("read failed")
	body := &failingBody{err: readErr}
	client := clientWithTransport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: body}, nil
	})
	_, err := client.Models(context.Background())
	if !errors.Is(err, readErr) || !body.closed {
		t.Fatalf("error = %v, body closed = %t", err, body.closed)
	}
}

func TestTransportMarshalErrorDoesNotSendRequest(t *testing.T) {
	calls := 0
	client := clientWithTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return testResponse(http.StatusOK, `{}`), nil
	})
	_, err := client.SystemOne(context.Background(), Object{"bad": make(chan int)}, Questions{"q": &Noul{Instructions: String("Question?")}})
	var unsupported *json.UnsupportedTypeError
	if !errors.As(err, &unsupported) || calls != 0 {
		t.Fatalf("error = %v, calls = %d; want marshal error before HTTP", err, calls)
	}
}

func TestTransportTimeoutDuringRequest(t *testing.T) {
	client := clientWithTransport(func(req *http.Request) (*http.Response, error) {
		if _, ok := req.Context().Deadline(); !ok {
			t.Error("request context has no deadline")
		}
		<-req.Context().Done()
		return nil, req.Context().Err()
	})
	client.timeout = 50 * time.Millisecond
	_, err := client.Models(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error = %v, want deadline exceeded", err)
	}
}

func TestTransportTimeoutDuringRetryWait(t *testing.T) {
	calls := 0
	client := clientWithTransport(func(*http.Request) (*http.Response, error) {
		calls++
		response := testResponse(http.StatusTooManyRequests, `{"detail":"rate limited"}`)
		response.Header.Set("Retry-After", "1")
		return response, nil
	})
	client.timeout = 100 * time.Millisecond
	client.maxRetries = 1
	_, err := client.Models(context.Background())
	if !errors.Is(err, context.DeadlineExceeded) || calls != 1 {
		t.Fatalf("error = %v, calls = %d; want deadline exceeded during first retry wait", err, calls)
	}
}

func TestRetryAfterSeconds(t *testing.T) {
	client := defaultClient()
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"2", 2 * time.Second},
		{"0", 0},
		{"-1", 0},
		{"not-a-number", 0},
		{"", 0},
	}
	for _, tt := range tests {
		if got := client.retryAfter(tt.input); got != tt.want {
			t.Errorf("retryAfter(%q) = %s, want %s", tt.input, got, tt.want)
		}
	}
}
