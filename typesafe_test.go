package typesafe

import (
	"net/http"
	"testing"
	"time"
)

func TestDefaultClient(t *testing.T) {
	c := defaultClient()
	if c.baseURL != defaultBaseURL || c.model != defaultModel || c.apiKey != "" {
		t.Fatalf("unexpected client defaults: baseURL=%q model=%q apiKey=%q", c.baseURL, c.model, c.apiKey)
	}
	if c.httpClient != http.DefaultClient || c.timeout != defaultTimeout || c.maxRetries != defaultMaxRetries {
		t.Fatalf("unexpected transport defaults: httpClient=%p timeout=%s maxRetries=%d", c.httpClient, c.timeout, c.maxRetries)
	}
}

func TestNewClientEnvironmentAndExplicitOptions(t *testing.T) {
	t.Setenv("TYPESAFE_BASE_URL", "https://env.test")
	t.Setenv("TYPESAFE_MODEL", "env-model")
	t.Setenv("TYPESAFE_API_KEY", "env-key")

	fromEnv := NewClient()
	if fromEnv.baseURL != "https://env.test" || fromEnv.model != "env-model" || fromEnv.apiKey != "env-key" {
		t.Fatalf("environment options not applied: baseURL=%q model=%q apiKey=%q", fromEnv.baseURL, fromEnv.model, fromEnv.apiKey)
	}

	httpClient := &http.Client{}
	overridden := NewClient(
		WithBaseURL("https://option.test"),
		WithModel("option-model"),
		WithApiKey("option-key"),
		WithHTTPClient(httpClient),
		WithTimeout(2*time.Second),
		WithMaxRetries(1),
	)
	if overridden.baseURL != "https://option.test" || overridden.model != "option-model" || overridden.apiKey != "option-key" {
		t.Fatalf("explicit options did not override environment: baseURL=%q model=%q apiKey=%q", overridden.baseURL, overridden.model, overridden.apiKey)
	}
	if overridden.httpClient != httpClient || overridden.timeout != 2*time.Second || overridden.maxRetries != 1 {
		t.Fatalf("explicit transport options not applied: httpClient=%p timeout=%s maxRetries=%d", overridden.httpClient, overridden.timeout, overridden.maxRetries)
	}
}
