package typesafe

import "encoding/json"

var (
	_ Question = (*Choice)(nil)
	_ Question = (*Score)(nil)
	_ Question = (*Noul)(nil)
	_ Question = (*Box)(nil)
)

type QuestionType string

func (qt QuestionType) String() string {
	return string(qt)
}

const (
	QuestionTypeChoice QuestionType = "choice"
	QuestionTypeScore  QuestionType = "score"
	QuestionTypeNoul   QuestionType = "noul"
	QuestionTypeBox    QuestionType = "box"
)

type Question interface {
	T() QuestionType // choice score noul box
}

type Questions map[string]Question

type Choice struct {
	Instructions Pandora `json:"instructions"`
	Criteria     Object  `json:"criteria"`
}

func (Choice) T() QuestionType { return QuestionTypeChoice }

func (c Choice) MarshalJSON() ([]byte, error) {
	type payload Choice
	return json.Marshal(struct {
		Type QuestionType `json:"type"`
		*payload
	}{c.T(), (*payload)(&c)})
}

type Score struct {
	Instructions Pandora `json:"instructions"`
	Criteria     Array   `json:"criteria"`
}

func (Score) T() QuestionType { return QuestionTypeScore }

func (s Score) MarshalJSON() ([]byte, error) {
	type payload Score
	return json.Marshal(struct {
		Type QuestionType `json:"type"`
		*payload
	}{s.T(), (*payload)(&s)})
}

type Noul struct {
	Instructions Pandora `json:"instructions"`
	Criteria     Object  `json:"criteria,omitempty"`
}

func (Noul) T() QuestionType { return QuestionTypeNoul }

func (n Noul) MarshalJSON() ([]byte, error) {
	type payload Noul
	return json.Marshal(struct {
		Type QuestionType `json:"type"`
		*payload
	}{n.T(), (*payload)(&n)})
}

type Box struct {
	Instructions Pandora `json:"instructions"`
}

func (Box) T() QuestionType { return QuestionTypeBox }

func (b Box) MarshalJSON() ([]byte, error) {
	type payload Box
	return json.Marshal(struct {
		Type QuestionType `json:"type"`
		*payload
	}{b.T(), (*payload)(&b)})
}
