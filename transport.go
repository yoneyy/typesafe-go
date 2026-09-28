package typesafe

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type ApiError struct {
	StatusCode int
	Detail     json.RawMessage
	Body       []byte
}

func (e ApiError) Error() string {
	if len(e.Detail) > 0 {
		return fmt.Sprintf("typesafe: http %d: %s", e.StatusCode, e.Detail)
	}
	return fmt.Sprintf("typesafe: http %d: %s", e.StatusCode, e.Body)
}

func (c *Client) doRaw(ctx context.Context, method string, path string, body any) ([]byte, error) {
	if c.timeout > 0 && ctx != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	var bodyByte []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyByte = b
	}

	up, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	up = up.JoinPath(path)

	for attempt := 0; ; attempt++ {
		var rd io.Reader
		if bodyByte != nil {
			rd = bytes.NewReader(bodyByte)
		}
		req, err := http.NewRequestWithContext(ctx, method, up.String(), rd)
		if err != nil {
			return nil, fmt.Errorf("typesafe: built request: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
		if bodyByte != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			reqErr := fmt.Errorf("typesafe: request: %w", err)
			if !c.retryWait(ctx, attempt, 0) {
				return nil, reqErr
			}
			continue
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("typesafe: read response body: %w", err)
		}

		if resp.StatusCode < http.StatusOK ||
			resp.StatusCode >= http.StatusMultipleChoices {
			apiErr := ApiError{
				StatusCode: resp.StatusCode,
				Body:       respBody,
			}
			var parsed struct {
				Detail json.RawMessage `json:"detail"`
			}
			if err := json.Unmarshal(respBody, &parsed); err == nil {
				apiErr.Detail = parsed.Detail
			}
			if resp.StatusCode == http.StatusTooManyRequests ||
				resp.StatusCode >= http.StatusInternalServerError {
				retryAfter := resp.Header.Get("Retry-After")
				if !c.retryWait(ctx, attempt, c.retryAfter(retryAfter)) {
					if err := ctx.Err(); err != nil {
						return nil, fmt.Errorf("typesafe: retry: %w", err)
					}
					return nil, apiErr
				}
				continue
			}
			return nil, apiErr
		}

		return respBody, nil
	}
}

func (c *Client) do(ctx context.Context, method string, path string, body, result any) error {
	resp, err := c.doRaw(ctx, method, path, body)
	if err != nil {
		return err
	}
	if result != nil {
		if err := json.Unmarshal(resp, result); err != nil {
			return fmt.Errorf("typesafe: unmarshal response: %w", err)
		}
	}
	return nil
}

func (c *Client) retryWait(ctx context.Context, attempt int, wait time.Duration) bool {
	if attempt >= c.maxRetries {
		return false
	}
	if wait <= 0 {
		wait = time.Duration(200*(1<<attempt)) * time.Millisecond
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(wait):
		return true
	}
}

func (c *Client) retryAfter(v string) time.Duration {
	secs, err := strconv.Atoi(v)
	if err != nil || secs < 0 {
		return 0
	}
	return time.Duration(secs) * time.Second
}
