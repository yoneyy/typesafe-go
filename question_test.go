package typesafe

import (
	"encoding/json"
	"testing"
)

func TestQuestionMarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		q    Question
		want string
	}{
		{
			name: "choice with string instructions",
			q: &Choice{
				Instructions: String("Choose a team"),
				Criteria:     Object{"platform": "Infrastructure", "product": nil},
			},
			want: `{"type":"choice","instructions":"Choose a team","criteria":{"platform":"Infrastructure","product":null}}`,
		},
		{
			name: "choice value",
			q: Choice{
				Instructions: String("Choose a team"),
				Criteria:     Object{"platform": "Infrastructure", "product": nil},
			},
			want: `{"type":"choice","instructions":"Choose a team","criteria":{"platform":"Infrastructure","product":null}}`,
		},
		{
			name: "score with object instructions",
			q: &Score{
				Instructions: Object{"question": "Rate impact", "context": []string{"checkout", "payments"}},
				Criteria:     Array{String("Low"), Object{"level": "High"}},
			},
			want: `{"type":"score","instructions":{"question":"Rate impact","context":["checkout","payments"]},"criteria":["Low",{"level":"High"}]}`,
		},
		{
			name: "score value",
			q: Score{
				Instructions: String("Rate impact"),
				Criteria:     Array{String("Low"), String("High")},
			},
			want: `{"type":"score","instructions":"Rate impact","criteria":["Low","High"]}`,
		},
		{
			name: "noul with array instructions and omitted criteria",
			q: &Noul{
				Instructions: Array{String("Is this urgent?"), Object{"policy": "Escalate outages"}},
			},
			want: `{"type":"noul","instructions":["Is this urgent?",{"policy":"Escalate outages"}]}`,
		},
		{
			name: "noul value with omitted criteria",
			q:    Noul{Instructions: String("Is this urgent?")},
			want: `{"type":"noul","instructions":"Is this urgent?"}`,
		},
		{
			name: "noul with criteria",
			q: &Noul{
				Instructions: String("Is this urgent?"),
				Criteria:     Object{"true": "Needs action", "false": "Can wait"},
			},
			want: `{"type":"noul","instructions":"Is this urgent?","criteria":{"true":"Needs action","false":"Can wait"}}`,
		},
		{
			name: "box without criteria",
			q:    &Box{Instructions: String("the coffee mug in photo")},
			want: `{"type":"box","instructions":"the coffee mug in photo"}`,
		},
		{
			name: "box value with object instructions",
			q:    Box{Instructions: Object{"target": "the person matching 9a21c7d3-badge", "image": "photo"}},
			want: `{"type":"box","instructions":{"target":"the person matching 9a21c7d3-badge","image":"photo"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(Questions{"question": tt.q})
			if err != nil {
				t.Fatal(err)
			}
			assertJSONEqual(t, `{"question":`+tt.want+`}`, got)
		})
	}
}

func TestPandoraTypedCollectionsMarshalJSON(t *testing.T) {
	tests := []struct {
		name  string
		value Pandora
		want  string
	}{
		{"typed array", TArray[string]{"first", "second"}, `["first","second"]`},
		{"typed object", TObject[string, int]{"count": 2}, `{"count":2}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.value)
			if err != nil {
				t.Fatal(err)
			}
			assertJSONEqual(t, tt.want, got)
		})
	}
}

func TestQuestionTypes(t *testing.T) {
	tests := []struct {
		question Question
		want     QuestionType
	}{
		{Choice{}, QuestionTypeChoice},
		{&Choice{}, QuestionTypeChoice},
		{Score{}, QuestionTypeScore},
		{&Score{}, QuestionTypeScore},
		{Noul{}, QuestionTypeNoul},
		{&Noul{}, QuestionTypeNoul},
		{Box{}, QuestionTypeBox},
		{&Box{}, QuestionTypeBox},
	}
	for _, tt := range tests {
		if got := tt.question.T(); got != tt.want || got.String() != string(tt.want) {
			t.Errorf("question type = %q, want %q", got, tt.want)
		}
	}
}
