package typesafe

import (
	"net/http"
	"time"
)

type ClientOptions func(*Client)

func WithBaseURL(baseURL string) ClientOptions {
	return func(c *Client) {
		c.baseURL = baseURL
	}
}

func WithModel(model string) ClientOptions {
	return func(c *Client) {
		c.model = model
	}
}

func WithApiKey(key string) ClientOptions {
	return func(c *Client) {
		c.apiKey = key
	}
}

func WithHTTPClient(client *http.Client) ClientOptions {
	return func(c *Client) {
		c.httpClient = client
	}
}

func WithTimeout(timeout time.Duration) ClientOptions {
	return func(c *Client) {
		c.timeout = timeout
	}
}

func WithMaxRetries(maxRetries int) ClientOptions {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}
