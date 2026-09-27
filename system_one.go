package typesafe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

var (
	_ Answer = (*ChoiceAnswer)(nil)
	_ Answer = (*ScoreAnswer)(nil)
	_ Answer = (*NoulAnswer)(nil)
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

type systemOneRequest struct {
	State     Pandora   `json:"state"`
	Model     string    `json:"model"`
	Questions Questions `json:"questions"`
}

type SystemOneOption func(*systemOneRequest)

func WithRequestModel(model string) SystemOneOption {
	return func(r *systemOneRequest) {
		r.Model = model
	}
}

const SystemOnePath = "/systemone"

func (c *Client) SystemOne(ctx context.Context, state Pandora, questions Questions, opts ...SystemOneOption) (*SystemOneResponse, error) {
	req := systemOneRequest{
		State:     state,
		Model:     c.model,
		Questions: questions,
	}
	for _, opt := range opts {
		opt(&req)
	}

	res, err := c.doRaw(ctx, http.MethodPost, SystemOnePath, req)
	if err != nil {
		return nil, err
	}
	var resp SystemOneResponse
	if err := json.Unmarshal(res, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
