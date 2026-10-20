# typesafe-go

[English](README.md) · [简体中文](README_zh.md)

An independent Go SDK for [TypeSafe AI](https://typesafe.ai/). Send text, structured data, or images to System One, ask typed questions with Jev, and work with typed answers in Go. The package also supports model discovery.

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
| `Box` | Find where something is in an image | None; see [Images and Box questions](#images-and-box-questions) |

`Choice`, `Score`, `Noul`, and `Box` can each be passed as a value or a pointer. A value is copied when added to `Questions`; use a pointer if later changes to the original struct should be reflected in the request.

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
	"impact": typesafe.Score{
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

Here, `client` is a `*typesafe.Client` and `ctx` is a `context.Context`. The `Choices()`, `Scores()`, `Nouls()`, and `Boxes()` helpers return maps containing only answers of that type. You can also inspect `response.Answers` directly.

The `state` and each question's `Instructions` accept `String`, `Object`, or `Array`. For structured instructions, put the question and its supporting data in an `Object`:

```go
instructions := typesafe.Object{
	"question": "Does this incident match the escalation policy?",
	"policy":   "Escalate if checkout is unavailable to customers.",
}
```

`TArray[V]` and `TObject[K, V]` are also available for typed collections, provided their elements and map keys can be encoded as JSON. See the [TypeSafe API reference](https://docs.typesafe.ai/api) for the request schema and criteria limits.

## Images and Box questions

> Images and Box questions are [dreamer-ready](https://docs.typesafe.ai/images): experimental features meant for prototypes. The request format, answers, and limits can change between releases.

Every request that contains an image or a Box question must send the `TypeSafe-Danger-Zone-Images: true` header. The SDK does not add it for you; pass `WithImages()`. Box questions also need Jev 1.14, so select `ModelJevPreview` (`jev-preview`) or `jev-1.14.0`.

Each image is listed once in `Assets`, keyed by a reference string. Write that reference wherever the image belongs in the state, instructions, or criteria, as a whole value or inside a longer string. The API matches references as plain text, so use references that cannot occur in your text by accident; `NewAssetRef()` returns a random one.

```go
photo, err := os.ReadFile("kitchen.jpg")
if err != nil {
	log.Fatal(err)
}
ref := typesafe.NewAssetRef()

response, err := client.SystemOne(ctx,
	typesafe.Object{"photo": ref},
	typesafe.Questions{
		"mug_visible": typesafe.Noul{Instructions: typesafe.String("Is there a coffee mug in `photo`?")},
		"mug":         typesafe.Box{Instructions: typesafe.String("the coffee mug in `photo`")},
	},
	typesafe.WithRequestModel(typesafe.ModelJevPreview),
	typesafe.WithAssets(typesafe.Assets{ref: typesafe.Image{Data: photo}}),
	typesafe.WithImages(),
)
if err != nil {
	log.Fatal(err)
}

mug := response.Boxes()["mug"]
fmt.Println(mug.Box.Top, mug.Box.Bottom, mug.Box.Left, mug.Box.Right)
```

A Box answer gives four edges as fractions of the image size, from the top-left corner (0, 0) to the bottom-right corner (1, 1). `Box.Pixels(width, height)` converts them to an `image.Rectangle` that can be passed to `SubImage`. `Confidence` is a `*float64` and is `nil` when the model does not report it. A Box question always returns a box, even when the target is absent, so pair it with a `Noul` as above.

Images can also go in a question's `Instructions` or `Criteria`, for example ``typesafe.String("Is the person in `door` the person in this badge photo: " + badgeRef + "?")``.

**Image data.** `Image.Data` holds PNG or JPEG file bytes. `Format` (`ImageFormatPNG`, `ImageFormatJPG`, or `ImageFormatJPEG`) is detected from the bytes when left empty. Use `ImageFromBase64` for base64 text. Images must be still PNG or JPEG files with each side 280–1568 pixels and at most 16 MiB; the whole request is limited to 32 MiB. See [Images](https://docs.typesafe.ai/images#limits) for all limits.

**Upload modes.** By default, requests with assets are sent as `multipart/form-data`, with each image as a file part; this is the mode TypeSafe recommends. Pass `WithBase64Assets()` to send the images base64-encoded inside a JSON body instead. Base64 makes each image about a third larger, and the image's format must be detectable or set in `Format`.

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
	typesafe.WithRequestModel(typesafe.ModelJevPreview))
```

`SystemOne` accepts these per-request options:

| Option | Purpose |
| --- | --- |
| `WithRequestModel` | Model for this request; `ModelJevLatest` and `ModelJevPreview` name the aliases |
| `WithAssets` | Images referenced by the request; calling it more than once merges the assets |
| `WithBase64Assets` | Send assets base64-encoded in JSON instead of as multipart file parts |
| `WithImages` | Send `TypeSafe-Danger-Zone-Images: true`, required for images and Box questions |
| `WithLongContext` | Send `TypeSafe-Danger-Zone-Long-Context: true`, raising both context limits to 262k tokens |
| `WithRequestHeader` | Send any other HTTP header with this request |

Long context is slower and less accurate than staying within the default limits, so use it only for requests that need it. See [Long context](https://docs.typesafe.ai/models#long-context).

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
- [Images](https://docs.typesafe.ai/images) and [Box](https://docs.typesafe.ai/primitives/box)

For the same guide in Chinese, see [README_zh.md](README_zh.md).
