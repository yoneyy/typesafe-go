# typesafe-go

[English](README.md) · [简体中文](README_zh.md)

[TypeSafe AI](https://typesafe.ai/) 的非官方 Go SDK。你可以把文本、结构化数据或图片交给 System One，使用 Jev 提出不同类型的问题，并以 Go 类型读取答案；也可以查询可用模型。

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
| `Box` | 找出某个目标在图片中的位置 | 无；见[图片与 Box 问题](#图片与-box-问题) |

`Choice`、`Score`、`Noul` 和 `Box` 都可以使用值或指针。放入 `Questions` 时，值会被复制；如果希望之后修改原结构体的字段时也更新请求中的问题，请使用指针。

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
	"impact": typesafe.Score{
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

这里的 `client` 是 `*typesafe.Client`，`ctx` 是 `context.Context`。`Choices()`、`Scores()`、`Nouls()` 和 `Boxes()` 分别返回对应类型的答案映射；也可以直接读取 `response.Answers`。

`state` 和每个问题的 `Instructions` 都支持 `String`、`Object` 和 `Array`。如果问题需要附带上下文，可以把问题和数据放在同一个 `Object` 中：

```go
instructions := typesafe.Object{
	"question": "这次事故是否符合升级处理规则？",
	"policy":   "如果用户无法完成结账，则需要升级处理。",
}
```

对于需要明确元素类型的集合，也可使用 `TArray[V]` 和 `TObject[K, V]`；其中的元素和映射键仍需能够编码为 JSON。请求结构和各类问题的限制见 [TypeSafe API 文档](https://docs.typesafe.ai/api)。

## 图片与 Box 问题

> 图片和 Box 问题目前是 [Dreamer-ready](https://docs.typesafe.ai/images) 状态，属于实验性功能，适合做原型。请求格式、答案和各项限制都可能随版本变化。

只要请求中包含图片或 Box 问题，就必须带上 `TypeSafe-Danger-Zone-Images: true` 请求头。SDK 不会自动添加，请传入 `WithImages()`。Box 问题还需要 Jev 1.14，请选择 `ModelJevPreview`（`jev-preview`）或 `jev-1.14.0`。

每张图片在 `Assets` 中登记一次，键是一个引用字符串。把这个引用写在 state、instructions 或 criteria 中需要图片的位置即可，可以是整个值，也可以嵌在一段文字中间。API 按纯文本匹配引用，所以引用不能和正文意外重合；`NewAssetRef()` 会生成一个随机引用。

```go
photo, err := os.ReadFile("kitchen.jpg")
if err != nil {
	log.Fatal(err)
}
ref := typesafe.NewAssetRef()

response, err := client.SystemOne(ctx,
	typesafe.Object{"photo": ref},
	typesafe.Questions{
		"mug_visible": typesafe.Noul{Instructions: typesafe.String("`photo` 中有咖啡杯吗？")},
		"mug":         typesafe.Box{Instructions: typesafe.String("`photo` 中的咖啡杯")},
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

Box 答案用四条边表示位置，每条边都是相对图片尺寸的比例：左上角为 (0, 0)，右下角为 (1, 1)。`Box.Pixels(width, height)` 会把它换算成 `image.Rectangle`，可以直接传给 `SubImage` 裁剪。`Confidence` 的类型是 `*float64`，模型没有给出置信度时为 `nil`。即使图片里没有目标，Box 问题也总会返回一个框，所以建议像上面那样同时问一个 `Noul`。

图片也可以放在问题的 `Instructions` 或 `Criteria` 中，例如 ``typesafe.String("`door` 中的人是这张工牌照片上的人吗：" + badgeRef + "？")``。

**图片数据。** `Image.Data` 存放 PNG 或 JPEG 文件的字节。`Format`（`ImageFormatPNG`、`ImageFormatJPG` 或 `ImageFormatJPEG`）留空时会根据文件内容自动识别。如果手上是 base64 文本，可以用 `ImageFromBase64`。图片必须是静态 PNG 或 JPEG，每条边 280–1568 像素，单个文件不超过 16 MiB，整个请求不超过 32 MiB。完整限制见 [Images](https://docs.typesafe.ai/images#limits)。

**上传方式。** 带图片的请求默认以 `multipart/form-data` 发送，每张图片是一个文件分段，这也是 TypeSafe 推荐的方式。传入 `WithBase64Assets()` 则改为把图片 base64 编码后放进 JSON 请求体。base64 会让每张图片增大约三分之一，而且图片格式必须能被识别，或者在 `Format` 中显式指定。

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
	typesafe.WithRequestModel(typesafe.ModelJevPreview))
```

`SystemOne` 支持以下单次请求选项：

| 选项 | 作用 |
| --- | --- |
| `WithRequestModel` | 本次请求使用的模型；`ModelJevLatest` 和 `ModelJevPreview` 是两个别名常量 |
| `WithAssets` | 请求中引用的图片；多次调用会合并 |
| `WithBase64Assets` | 把图片 base64 编码后放进 JSON，而不是作为 multipart 文件分段发送 |
| `WithImages` | 发送 `TypeSafe-Danger-Zone-Images: true`，使用图片和 Box 问题时必须加 |
| `WithLongContext` | 发送 `TypeSafe-Danger-Zone-Long-Context: true`，把两项上下文上限都提高到 262k token |
| `WithRequestHeader` | 为本次请求添加任意其他 HTTP 请求头 |

长上下文比默认上限更慢，准确率也更低，只在确实需要时开启。详见 [Long context](https://docs.typesafe.ai/models#long-context)。

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
- [Images](https://docs.typesafe.ai/images) 与 [Box](https://docs.typesafe.ai/primitives/box)

英文版请见 [README.md](README.md)。
