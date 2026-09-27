# typesafe-go

[English](README.md) · [简体中文](README_zh.md)

[TypeSafe AI](https://typesafe.ai/) 的非官方 Go SDK。你可以把文本或结构化数据交给 System One，使用 Jev 提出不同类型的问题，并以 Go 类型读取答案；也可以查询可用模型。

本项目由社区维护，与 TypeSafe AI 没有关联。

## 安装

```sh
go get github.com/yoneyy/typesafe-go
```

## 快速开始

先设置 API Key。`NewClient` 会自动从环境变量读取。

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
		typesafe.String("最新部署后，结账请求的错误率上升了。"),
		typesafe.Questions{
			"needs_attention": &typesafe.Noul{
				Instructions: typesafe.String("这是否需要立即处理？"),
			},
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	answer, ok := response.Nouls()["needs_attention"]
	if !ok {
		log.Fatal("缺少 needs_attention 答案")
	}
	fmt.Printf("noul: %.3f; model: %s; input tokens: %d\n",
		answer.Noul, response.Model, response.Usage.InputTokens)
}
```

`Noul` 返回 0 到 1 之间的值，表示“是”的概率。响应中还包含实际使用的模型 ID 和 token 用量。

## 问题与答案

`SystemOne` 接收 `Questions` 映射。问题的键由你指定，响应会用相同的键返回答案。SDK 会自动给问题的 JSON 添加 `type`，无需手动填写。

| 问题类型 | 用途 | `Criteria` |
| --- | --- | --- |
| `Choice` | 从命名选项中选出一个 | 必填 `Object`，描述各选项；最多 255 项 |
| `Score` | 按有序等级评分 | 必填 `Array`，包含 2–10 个等级描述 |
| `Noul` | 提出是/否问题 | 可选 `Object`，描述 `true` 和 `false` |

一次调用可以针对同一份状态提出不同类型的问题：

```go
state := typesafe.Object{
	"service":  "checkout",
	"incident": "部署后请求开始超时",
}

questions := typesafe.Questions{
	"owner": &typesafe.Choice{
		Instructions: typesafe.String("应该由哪个团队排查？"),
		Criteria: typesafe.Object{
			"platform": "基础设施与网络",
			"product":  "应用程序行为",
		},
	},
	"impact": &typesafe.Score{
		Instructions: typesafe.String("对用户的影响有多严重？"),
		Criteria:     typesafe.Array{"轻微", "中等", "严重"},
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

这里的 `client` 是 `*typesafe.Client`，`ctx` 是 `context.Context`。`Choices()`、`Scores()` 和 `Nouls()` 分别返回对应类型的答案映射；也可以直接读取 `response.Answers`。

`state` 和每个问题的 `Instructions` 都支持 `String`、`Object` 和 `Array`。如果问题需要附带上下文，可以把问题和数据放在同一个 `Object` 中：

```go
instructions := typesafe.Object{
	"question": "这次事故是否符合升级处理规则？",
	"policy":   "如果用户无法完成结账，则需要升级处理。",
}
```

对于需要明确元素类型的集合，也可使用 `TArray[V]` 和 `TObject[K, V]`；其中的元素和映射键仍需能够编码为 JSON。请求结构和各类问题的限制见 [TypeSafe API 文档](https://docs.typesafe.ai/api)。

## 配置

`NewClient()` 先使用包内默认值，再应用环境变量，最后应用显式传入的选项；因此显式选项优先级最高。

| 选项 | 环境变量 | 作用 |
| --- | --- | --- |
| `WithApiKey` | `TYPESAFE_API_KEY` | API Key |
| `WithModel` | `TYPESAFE_MODEL` | 默认模型；未配置时为 `jev-latest` |
| `WithBaseURL` | `TYPESAFE_BASE_URL` | API 基础地址 |
| `WithTimeout` | — | 整次请求（包含重试）的超时时间；默认 1 分钟 |
| `WithMaxRetries` | — | 额外重试次数；默认 3 次 |
| `WithHTTPClient` | — | 自定义 `*http.Client` |

```go
client := typesafe.NewClient(
	typesafe.WithApiKey("your-api-key"),
	typesafe.WithModel("jev-latest"),
	typesafe.WithTimeout(30*time.Second),
)
```

如果只想为某一次评估指定模型，可以给 `SystemOne` 传入 `WithRequestModel`：

```go
response, err := client.SystemOne(ctx, state, questions,
	typesafe.WithRequestModel("jev-latest"))
```

## 模型与错误处理

通过 `Models` 查询当前账号可用的模型：

```go
models, err := client.Models(ctx)
if err != nil {
	log.Fatal(err)
}
for _, model := range models {
	fmt.Println(model.Name, model.ReleaseDate)
}
```

非 2xx HTTP 响应会返回 `typesafe.ApiError`，其中包含 `StatusCode`、原始响应 `Body`，以及 API 提供时的 `Detail`。客户端会按 `WithMaxRetries` 的配置重试传输错误、HTTP 429 和 5xx；如果响应带有秒数形式的 `Retry-After`，也会按其等待。

```go
var apiErr typesafe.ApiError
if errors.As(err, &apiErr) {
	fmt.Printf("API 状态码 %d：%s\n", apiErr.StatusCode, apiErr.Detail)
}
```

## 参考资料

- [TypeSafe API 文档](https://docs.typesafe.ai/api)
- [TypeSafe 模型文档](https://docs.typesafe.ai/models)

英文版请见 [README.md](README.md)。
