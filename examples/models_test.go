package main

import (
	"testing"

	"github.com/yoneyy/typesafe-go"
)

func TestModels(t *testing.T) {
	cli := typesafe.NewClient()
	models, err := cli.Models(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v\n", models)
}
