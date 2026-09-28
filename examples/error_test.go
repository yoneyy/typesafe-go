package main

import (
	"errors"
	"testing"

	"github.com/yoneyy/typesafe-go"
)

func TestApiError(t *testing.T) {
	cli := typesafe.NewClient(typesafe.WithApiKey("example_api_key"))
	res, err := cli.SystemOne(
		t.Context(),
		typesafe.String("Hi, I've been trying to connect my Stripe account for 3 days and the integration keeps failing. I'm losing sales. Please help ASAP"),
		typesafe.Questions{
			"urgency": typesafe.Noul{
				Instructions: typesafe.String("Does this message express urgency?"),
				//Criteria:     nil,
			},
		},
	)
	if err != nil {
		// https://docs.typesafe.ai/api#errors
		if apiErr, ok := errors.AsType[typesafe.ApiError](err); ok {
			t.Logf("Error: %+v\n", apiErr)           // Error: typesafe: http 403: {"error_type":"authentication_error","message":"Must supply an API key! Check your request and try again."}
			t.Logf("Error: %s\n", apiErr.Detail)     // Error: {"error_type":"authentication_error","message":"Must supply an API key! Check your request and try again."}
			t.Logf("Error: %s\n", apiErr.Body)       // Error: {"detail":{"error_type":"authentication_error","message":"Must supply an API key! Check your request and try again."}}
			t.Logf("Error: %d\n", apiErr.StatusCode) // Error: 403
		}
		return
	}

	t.Logf("%+v\n", res)
}
