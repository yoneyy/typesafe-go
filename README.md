# typesafe-go

[English](README.md) · [简体中文](README_zh.md)

An independent Go SDK for [TypeSafe AI](https://typesafe.ai/). Send text or structured data to System One, ask typed questions with Jev, and work with typed answers in Go. The package also supports model discovery.

This is a community project and is not affiliated with TypeSafe AI.

## Install

```sh
go get github.com/yoneyy/typesafe-go
```

## Quick start

Set your API key before running the example. `NewClient` reads it from the environment automatically.

```sh
export TYPESAFE_API_KEY="your-api-key"
```

```go
package main

import (
	"context"
	"fmt"
	"log"

	typesafe "github.com/yoneyy/typesafe-go"
)

func main() {
	client := typesafe.NewClient()

	response, err := client.SystemOne(
		context.Background(),
		typesafe.String("Checkout errors increased after the latest deployment."),
		typesafe.Questions{
			"needs_attention": &typesafe.Noul{
				Instructions: typesafe.String("Does this require immediate attention?"),
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	answer, ok := response.Nouls()["needs_attention"]
	if !ok {
		log.Fatal("missing needs_attention answer")
	}
	fmt.Printf("noul: %.3f; model: %s; input tokens: %d\n",
		answer.Noul, response.Model, response.Usage.InputTokens)
}
```

`Noul` returns a value from 0 to 1, representing the probability of a “yes” answer. The returned model ID and token usage are available on the response.

## Questions and answers

`SystemOne` accepts a `Questions` map. Each key is chosen by you and is used again for the corresponding answer. The SDK adds the question's JSON `type` automatically; you do not need to set it.

| Question | Use it for | `Criteria` |
| --- | --- | --- |
| `Choice` | Select one named option | Required `Object` of option descriptions; up to 255 options |
| `Score` | Rate against ordered levels | Required `Array` of 2–10 level descriptions |
| `Noul` | Ask a yes/no question | Optional `Object` describing `true` and `false` |

For example, a single call can ask different kinds of questions about the same state:

```go
state := typesafe.Object{
	"service": "checkout",
	"incident": "Requests are timing out after a deployment",
}

questions := typesafe.Questions{
	"owner": &typesafe.Choice{
		Instructions: typesafe.String("Which team should investigate?"),
		Criteria: typesafe.Object{
			"platform": "Infrastructure and networking",
			"product":  "Application behavior",
		},
	},
	"impact": &typesafe.Score{
		Instructions: typesafe.String("How severe is the customer impact?"),
		Criteria:     typesafe.Array{"Low", "Moderate", "High"},
	},
}

response, err := client.SystemOne(ctx, state, questions)
if err != nil {
	log.Fatal(err)
}

owner := response.Choices()["owner"]
impact := response.Scores()["impact"]
fmt.Println(owner.Choice, owner.Confidence, impact.Score)
```

Here, `client` is a `*typesafe.Client` and `ctx` is a `context.Context`. The `Choices()`, `Scores()`, and `Nouls()` helpers return maps containing only answers of that type. You can also inspect `response.Answers` directly.

The `state` and each question's `Instructions` accept `String`, `Object`, or `Array`. For structured instructions, put the question and its supporting data in an `Object`:

```go
instructions := typesafe.Object{
	"question": "Does this incident match the escalation policy?",
	"policy":   "Escalate if checkout is unavailable to customers.",
}
```

`TArray[V]` and `TObject[K, V]` are also available for typed collections, provided their elements and map keys can be encoded as JSON. See the [TypeSafe API reference](https://docs.typesafe.ai/api) for the request schema and criteria limits.

## Configuration

`NewClient()` starts with package defaults, then applies environment variables, then any options you pass explicitly. Explicit options therefore take precedence.

| Option | Environment variable | Purpose |
| --- | --- | --- |
| `WithApiKey` | `TYPESAFE_API_KEY` | API key |
| `WithModel` | `TYPESAFE_MODEL` | Default model; `jev-latest` unless changed |
| `WithBaseURL` | `TYPESAFE_BASE_URL` | API base URL |
| `WithTimeout` | — | Timeout for the whole request, including retries; default 1 minute |
| `WithMaxRetries` | — | Additional retries; default 3 |
| `WithHTTPClient` | — | Custom `*http.Client` |

```go
client := typesafe.NewClient(
	typesafe.WithApiKey("your-api-key"),
	typesafe.WithModel("jev-latest"),
	typesafe.WithTimeout(30*time.Second),
)
```

To select a model for one evaluation without changing the client default, pass `WithRequestModel` to `SystemOne`:

```go
response, err := client.SystemOne(ctx, state, questions,
	typesafe.WithRequestModel("jev-latest"))
```

## Models and errors

Use `Models` to list the models available to your account:

```go
models, err := client.Models(ctx)
if err != nil {
	log.Fatal(err)
}
for _, model := range models {
	fmt.Println(model.Name, model.ReleaseDate)
}
```

Non-2xx HTTP responses return `typesafe.ApiError`, with `StatusCode`, raw response `Body`, and `Detail` when the API supplies it. The client retries transport errors, HTTP 429, and 5xx responses according to `WithMaxRetries`; a numeric `Retry-After` header is honored when present.

```go
var apiErr typesafe.ApiError
if errors.As(err, &apiErr) {
	fmt.Printf("API status %d: %s\n", apiErr.StatusCode, apiErr.Detail)
}
```

## References

- [TypeSafe API reference](https://docs.typesafe.ai/api)
- [TypeSafe models](https://docs.typesafe.ai/models)

For the same guide in Chinese, see [README_zh.md](README_zh.md).
