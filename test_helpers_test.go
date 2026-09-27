package typesafe

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func clientWithTransport(transport roundTripFunc) *Client {
	c := defaultClient()
	c.baseURL = "https://example.test"
	c.apiKey = "test-key"
	c.httpClient = &http.Client{Transport: transport}
	c.maxRetries = 0
	return c
}

func testResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func assertJSONEqual(t *testing.T, want string, got []byte) {
	t.Helper()
	var wantValue, gotValue any
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("invalid expected JSON: %v", err)
	}
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("invalid actual JSON %q: %v", got, err)
	}
	if !reflect.DeepEqual(wantValue, gotValue) {
		t.Fatalf("JSON mismatch:\nwant %s\n got %s", want, got)
	}
}
