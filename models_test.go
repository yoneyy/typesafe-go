package typesafe

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

func TestModels(t *testing.T) {
	client := clientWithTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodGet {
			t.Errorf("method = %q, want GET", req.Method)
		}
		if req.Body != nil {
			t.Errorf("GET request unexpectedly has a body")
		}
		if got := req.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization = %q", got)
		}
		return testResponse(http.StatusOK, `{"models":[{"name":"jev-latest","description":"Current release","release_date":"2026-09-01"}]}`), nil
	})
	models, err := client.Models(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []Model{{Name: "jev-latest", Description: "Current release", ReleaseDate: "2026-09-01"}}
	if !reflect.DeepEqual(models, want) {
		t.Fatalf("Models() = %+v, want %+v", models, want)
	}
}

func TestModelsInvalidResponse(t *testing.T) {
	client := clientWithTransport(func(*http.Request) (*http.Response, error) {
		return testResponse(http.StatusOK, `{"models":{}}`), nil
	})
	models, err := client.Models(context.Background())
	if models != nil || err == nil || !strings.Contains(err.Error(), "typesafe: unmarshal response") {
		t.Fatalf("Models() = (%v, %v), want nil models and wrapped JSON error", models, err)
	}
}

func TestModelsAPIError(t *testing.T) {
	client := clientWithTransport(func(*http.Request) (*http.Response, error) {
		return testResponse(http.StatusUnauthorized, `{"detail":"invalid key"}`), nil
	})
	models, err := client.Models(context.Background())
	var apiErr ApiError
	if models != nil || !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("Models() = (%v, %v), want 401 ApiError", models, err)
	}
}
