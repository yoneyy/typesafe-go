package typesafe

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"maps"
	"math"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"slices"
)

var (
	_ Answer = (*ChoiceAnswer)(nil)
	_ Answer = (*ScoreAnswer)(nil)
	_ Answer = (*NoulAnswer)(nil)
	_ Answer = (*BoxAnswer)(nil)
)

type SystemOneResponse struct {
	Model   string            `json:"model"`
	Usage   Usage             `json:"usage"`
	Answers map[string]Answer `json:"answers"`
}

type AnswerType string

func (at AnswerType) String() string {
	return string(at)
}

const (
	AnswerTypeChoice AnswerType = "choice"
	AnswerTypeScore  AnswerType = "score"
	AnswerTypeNoul   AnswerType = "noul"
	AnswerTypeBox    AnswerType = "box"
)

type Answer interface {
	T() AnswerType
}
type Usage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type ChoiceAnswer struct {
	Type          AnswerType         `json:"type"`
	Choice        string             `json:"choice"`
	Confidence    float64            `json:"confidence"`
	Probabilities map[string]float64 `json:"probabilities"`
}

func (ChoiceAnswer) T() AnswerType {
	return AnswerTypeChoice
}

type ScoreAnswer struct {
	Type          AnswerType         `json:"type"`
	Score         float64            `json:"score"`
	Legend        map[string]string  `json:"legend"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    float64            `json:"confidence"`
}

func (ScoreAnswer) T() AnswerType {
	return AnswerTypeScore
}

type NoulAnswer struct {
	Type AnswerType `json:"type"`
	Noul float64    `json:"noul"`
}

func (NoulAnswer) T() AnswerType {
	return AnswerTypeNoul
}

type BoxCoordinates struct {
	Top    float64 `json:"top"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
	Right  float64 `json:"right"`
}

// Pixels converts the box edges, given as fractions of the image size, to a
// pixel rectangle for an image of width × height, clamped to the image.
func (b BoxCoordinates) Pixels(width, height int) image.Rectangle {
	edge := func(v float64, size int) int {
		return int(math.Round(min(max(v, 0), 1) * float64(size)))
	}
	return image.Rect(edge(b.Left, width), edge(b.Top, height), edge(b.Right, width), edge(b.Bottom, height))
}

type BoxAnswer struct {
	Type       AnswerType     `json:"type"`
	Box        BoxCoordinates `json:"box"`
	Confidence *float64       `json:"confidence"` // nil when the model does not provide it
}

func (BoxAnswer) T() AnswerType {
	return AnswerTypeBox
}

func (s *SystemOneResponse) UnmarshalJSON(data []byte) error {
	var r struct {
		Model   string                     `json:"model"`
		Usage   Usage                      `json:"usage"`
		Answers map[string]json.RawMessage `json:"answers"`
	}
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}
	s.Model = r.Model
	s.Usage = r.Usage
	s.Answers = make(map[string]Answer, len(r.Answers))

	for k, v := range r.Answers {
		var m struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(v, &m); err != nil {
			return err
		}
		var (
			answer Answer
			err    error
		)
		switch AnswerType(m.Type) {
		case AnswerTypeChoice:
			var a ChoiceAnswer
			err, answer = json.Unmarshal(v, &a), a
		case AnswerTypeScore:
			var a ScoreAnswer
			err, answer = json.Unmarshal(v, &a), a
		case AnswerTypeNoul:
			var a NoulAnswer
			err, answer = json.Unmarshal(v, &a), a
		case AnswerTypeBox:
			var a BoxAnswer
			err, answer = json.Unmarshal(v, &a), a
		default:
			return fmt.Errorf("%q unknown answer type %q", k, m.Type)
		}
		if err != nil {
			return fmt.Errorf("answer %q: %w", k, err)
		}
		s.Answers[k] = answer
	}
	return nil
}

func (s *SystemOneResponse) Choices() map[string]ChoiceAnswer {
	answers := make(map[string]ChoiceAnswer, len(s.Answers))
	for k, v := range s.Answers {
		if a, ok := v.(ChoiceAnswer); ok {
			answers[k] = a
		}
	}
	return answers
}

func (s *SystemOneResponse) Scores() map[string]ScoreAnswer {
	answers := make(map[string]ScoreAnswer, len(s.Answers))
	for k, v := range s.Answers {
		if a, ok := v.(ScoreAnswer); ok {
			answers[k] = a
		}
	}
	return answers
}

func (s *SystemOneResponse) Nouls() map[string]NoulAnswer {
	answers := make(map[string]NoulAnswer, len(s.Answers))
	for k, v := range s.Answers {
		if a, ok := v.(NoulAnswer); ok {
			answers[k] = a
		}
	}
	return answers
}

func (s *SystemOneResponse) Boxes() map[string]BoxAnswer {
	answers := make(map[string]BoxAnswer, len(s.Answers))
	for k, v := range s.Answers {
		if a, ok := v.(BoxAnswer); ok {
			answers[k] = a
		}
	}
	return answers
}

type systemOneRequest struct {
	State     Pandora                 `json:"state"`
	Model     string                  `json:"model"`
	Questions Questions               `json:"questions"`
	Assets    map[string]assetPayload `json:"assets,omitempty"`
}

type assetPayload struct {
	Type       AssetType   `json:"type"`
	Format     ImageFormat `json:"format,omitempty"`
	DataBase64 string      `json:"data_base64,omitempty"`
	Part       string      `json:"part,omitempty"`
}

type systemOneConfig struct {
	model  string
	assets Assets
	header http.Header
	base64 bool
}

type SystemOneOption func(*systemOneConfig)

func WithRequestModel(model string) SystemOneOption {
	return func(c *systemOneConfig) {
		c.model = model
	}
}

// WithAssets adds assets to the request. Calling it more than once merges the
// assets. Requests with assets need the TypeSafe-Danger-Zone-Images header.
func WithAssets(assets Assets) SystemOneOption {
	return func(c *systemOneConfig) {
		if c.assets == nil {
			c.assets = make(Assets, len(assets))
		}
		maps.Copy(c.assets, assets)
	}
}

// WithBase64Assets sends assets base64-encoded in a JSON body instead of as
// multipart/form-data file parts.
func WithBase64Assets() SystemOneOption {
	return func(c *systemOneConfig) {
		c.base64 = true
	}
}

func WithRequestHeader(key, value string) SystemOneOption {
	return func(c *systemOneConfig) {
		if c.header == nil {
			c.header = make(http.Header)
		}
		c.header.Set(key, value)
	}
}

const (
	HeaderDangerZoneImages      = "TypeSafe-Danger-Zone-Images"
	HeaderDangerZoneLongContext = "TypeSafe-Danger-Zone-Long-Context"
)

// WithImages turns on images and Box questions for the request.
func WithImages() SystemOneOption {
	return WithRequestHeader(HeaderDangerZoneImages, "true")
}

// WithLongContext raises the request's context limits.
func WithLongContext() SystemOneOption {
	return WithRequestHeader(HeaderDangerZoneLongContext, "true")
}

const SystemOnePath = "/systemone"

func (c *Client) SystemOne(ctx context.Context, state Pandora, questions Questions, opts ...SystemOneOption) (*SystemOneResponse, error) {
	cfg := systemOneConfig{model: c.model}
	for _, opt := range opts {
		opt(&cfg)
	}
	req := systemOneRequest{
		State:     state,
		Model:     cfg.model,
		Questions: questions,
	}

	body, contentType, err := cfg.encode(req)
	if err != nil {
		return nil, err
	}
	res, err := c.send(ctx, http.MethodPost, SystemOnePath, body, contentType, cfg.header)
	if err != nil {
		return nil, err
	}
	var resp SystemOneResponse
	if err := json.Unmarshal(res, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *systemOneConfig) encode(req systemOneRequest) ([]byte, string, error) {
	if len(c.assets) == 0 {
		body, err := json.Marshal(req)
		return body, "application/json", err
	}

	refs := slices.Sorted(maps.Keys(c.assets))
	images := make([]Image, len(refs))
	req.Assets = make(map[string]assetPayload, len(refs))
	for i, ref := range refs {
		img, err := assetImage(ref, c.assets[ref])
		if err != nil {
			return nil, "", err
		}
		images[i] = img
		if c.base64 {
			format := img.format()
			if format == "" {
				return nil, "", fmt.Errorf("typesafe: asset %q: unknown image format, set Image.Format", ref)
			}
			req.Assets[ref] = assetPayload{
				Type:       AssetTypeImage,
				Format:     format,
				DataBase64: base64.StdEncoding.EncodeToString(img.Data),
			}
		} else {
			req.Assets[ref] = assetPayload{Type: AssetTypeImage, Part: fmt.Sprintf("asset-%d", i)}
		}
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, "", err
	}
	if c.base64 {
		return reqJSON, "application/json", nil
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="request"`)
	h.Set("Content-Type", "application/json")
	part, err := w.CreatePart(h)
	if err != nil {
		return nil, "", err
	}
	if _, err := part.Write(reqJSON); err != nil {
		return nil, "", err
	}
	for i, ref := range refs {
		name := req.Assets[ref].Part
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, name, name))
		h.Set("Content-Type", images[i].format().contentType())
		part, err := w.CreatePart(h)
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(images[i].Data); err != nil {
			return nil, "", err
		}
	}
	if err := w.Close(); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), w.FormDataContentType(), nil
}
