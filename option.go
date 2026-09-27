package typesafe

import (
	"net/http"
	"time"
)

type ClientOption func(*Client)

func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

func WithModel(model string) ClientOption {
	return func(c *Client) {
		c.model = model
	}
}

func WithApiKey(key string) ClientOption {
	return func(c *Client) {
		c.apiKey = key
	}
}

func WithHTTPClient(client *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = client
	}
}

func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.timeout = timeout
	}
}

func WithMaxRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}
