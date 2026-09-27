package typesafe

import "encoding/json"

var (
	_ Question = (*Choice)(nil)
	_ Question = (*Score)(nil)
	_ Question = (*Noul)(nil)
)

type QuestionType string

func (qt QuestionType) String() string {
	return string(qt)
}

const (
	QuestionTypeChoice QuestionType = "choice"
	QuestionTypeScore  QuestionType = "score"
	QuestionTypeNoul   QuestionType = "noul"
)

type Question interface {
	T() QuestionType // choice score noul
}

type Questions map[string]Question

type questionBox[T any] struct {
	Type    QuestionType `json:"type"`
	Payload *T
}

type Choice struct {
	Instructions Pandora `json:"instructions"`
	Criteria     Object  `json:"criteria"`
}

func (*Choice) T() QuestionType { return QuestionTypeChoice }

func (c *Choice) MarshalJSON() ([]byte, error) {
	type payload Choice
	return json.Marshal(questionBox[payload]{
		Type:    c.T(),
		Payload: (*payload)(c),
	})
}

type Score struct {
	Instructions Pandora `json:"instructions"`
	Criteria     Array   `json:"criteria"`
}

func (*Score) T() QuestionType { return QuestionTypeScore }

func (s *Score) MarshalJSON() ([]byte, error) {
	type payload Score
	return json.Marshal(questionBox[payload]{
		Type:    s.T(),
		Payload: (*payload)(s),
	})
}

type Noul struct {
	Instructions Pandora `json:"instructions"`
	Criteria     Object  `json:"criteria"`
}

func (*Noul) T() QuestionType { return QuestionTypeNoul }

func (n *Noul) MarshalJSON() ([]byte, error) {
	type payload Noul
	return json.Marshal(questionBox[payload]{
		Type:    n.T(),
		Payload: (*payload)(n),
	})
}
