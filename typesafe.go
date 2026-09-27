package typesafe

import (
	"net/http"
	"os"
	"time"
)

const (
	defaultModel      = "jev-latest"
	defaultBaseURL    = "https://api.typesafe.ai/v1"
	defaultTimeout    = 1 * time.Minute
	defaultMaxRetries = 3
)

type Client struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
	timeout    time.Duration
	maxRetries int
}

func defaultClient() *Client {
	return &Client{
		baseURL:    defaultBaseURL,
		model:      defaultModel,
		apiKey:     "",
		httpClient: http.DefaultClient,
		timeout:    defaultTimeout,
		maxRetries: defaultMaxRetries,
	}
}

// DefaultClientOptions read from the environment(TYPESAFE_BASE_URL,
// TYPESAFE_MODEL, TYPESAFE_API_KEY).
// This should be used to initialize new clients.
func DefaultClientOptions() []ClientOption {
	opts := make([]ClientOption, 0)
	if o, ok := os.LookupEnv("TYPESAFE_BASE_URL"); ok {
		opts = append(opts, WithBaseURL(o))
	}
	if o, ok := os.LookupEnv("TYPESAFE_MODEL"); ok {
		opts = append(opts, WithModel(o))
	}
	if o, ok := os.LookupEnv("TYPESAFE_API_KEY"); ok {
		opts = append(opts, WithApiKey(o))
	}
	return opts
}

func NewClient(opts ...ClientOption) *Client {
	c := defaultClient()
	defaultOpts := DefaultClientOptions()
	opts = append(defaultOpts, opts...)
	for _, opt := range opts {
		opt(c)
	}
	return c
}
