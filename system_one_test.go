package typesafe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

const mixedAnswersJSON = `{
	"model":"jev-1.13.0",
	"usage":{"input_tokens":12,"output_tokens":4},
	"answers":{
		"owner":{"type":"choice","choice":"platform","confidence":0.8,"probabilities":{"platform":0.8,"product":0.2}},
		"impact":{"type":"score","score":1.5,"legend":{"0":"Low","1":"High"},"probabilities":{"0":0.25,"1":0.75},"confidence":0.7},
		"urgent":{"type":"noul","noul":0.9}
	}
}`

func TestSystemOneResponseDecodeAndRoundTrip(t *testing.T) {
	var response SystemOneResponse
	if err := json.Unmarshal([]byte(mixedAnswersJSON), &response); err != nil {
		t.Fatal(err)
	}
	if response.Model != "jev-1.13.0" || response.Usage != (Usage{InputTokens: 12, OutputTokens: 4}) {
		t.Fatalf("unexpected response metadata: model=%q usage=%+v", response.Model, response.Usage)
	}
	if len(response.Answers) != 3 || len(response.Choices()) != 1 || len(response.Scores()) != 1 || len(response.Nouls()) != 1 {
		t.Fatalf("unexpected answer counts: all=%d choices=%d scores=%d nouls=%d",
			len(response.Answers), len(response.Choices()), len(response.Scores()), len(response.Nouls()))
	}
	if got := response.Choices()["owner"]; got.Type != AnswerTypeChoice || got.Choice != "platform" || got.Probabilities["product"] != 0.2 {
		t.Errorf("choice answer = %+v", got)
	}
	if got := response.Scores()["impact"]; got.Type != AnswerTypeScore || got.Score != 1.5 || got.Legend["1"] != "High" {
		t.Errorf("score answer = %+v", got)
	}
	if got := response.Nouls()["urgent"]; got.Type != AnswerTypeNoul || got.Noul != 0.9 {
		t.Errorf("noul answer = %+v", got)
	}
	for key, want := range map[string]AnswerType{"owner": AnswerTypeChoice, "impact": AnswerTypeScore, "urgent": AnswerTypeNoul} {
		if got := response.Answers[key].T(); got != want || got.String() != string(want) {
			t.Errorf("answer %q type = %q, want %q", key, got, want)
		}
	}

	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, mixedAnswersJSON, encoded)
}

func TestSystemOneResponseRejectsInvalidAnswers(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{"invalid JSON", `{`, "unexpected end"},
		{"unknown type", `{"answers":{"q":{"type":"other"}}}`, `unknown answer type`},
		{"invalid answer field", `{"answers":{"q":{"type":"choice","confidence":"high"}}}`, `answer "q"`},
		{"non-object answer", `{"answers":{"q":"wrong"}}`, `cannot unmarshal string`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response SystemOneResponse
			err := json.Unmarshal([]byte(tt.body), &response)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("UnmarshalJSON error = %v, want substring %q", err, tt.want)
			}
		})
	}
}

func TestSystemOneRequestAndModelOverride(t *testing.T) {
	var requestBodies [][]byte
	client := clientWithTransport(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", req.Method)
		}
		if got := req.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Errorf("authorization = %q", got)
		}
		if got := req.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("content type = %q", got)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		requestBodies = append(requestBodies, body)
		return testResponse(http.StatusOK, mixedAnswersJSON), nil
	})
	client.model = "default-model"
	state := Object{"ticket": "Checkout timed out"}
	questions := Questions{"urgent": &Noul{Instructions: String("Is this urgent?")}}

	first, err := client.SystemOne(context.Background(), state, questions, WithRequestModel("override-model"))
	if err != nil {
		t.Fatal(err)
	}
	if first.Model != "jev-1.13.0" || first.Nouls()["urgent"].Noul != 0.9 {
		t.Fatalf("unexpected decoded response: %+v", first)
	}
	valueQuestions := Questions{"urgent": Noul{Instructions: String("Is this urgent?")}}
	if _, err := client.SystemOne(context.Background(), state, valueQuestions); err != nil {
		t.Fatal(err)
	}
	if client.model != "default-model" || len(requestBodies) != 2 {
		t.Fatalf("client model = %q, request count = %d", client.model, len(requestBodies))
	}
	assertJSONEqual(t, `{"state":{"ticket":"Checkout timed out"},"model":"override-model","questions":{"urgent":{"type":"noul","instructions":"Is this urgent?"}}}`, requestBodies[0])
	assertJSONEqual(t, `{"state":{"ticket":"Checkout timed out"},"model":"default-model","questions":{"urgent":{"type":"noul","instructions":"Is this urgent?"}}}`, requestBodies[1])
}

func TestSystemOneInvalidResponse(t *testing.T) {
	client := clientWithTransport(func(*http.Request) (*http.Response, error) {
		return testResponse(http.StatusOK, `not json`), nil
	})
	response, err := client.SystemOne(context.Background(), String("state"), Questions{"q": &Noul{Instructions: String("Question?")}})
	if err == nil || response != nil {
		t.Fatalf("SystemOne() = (%v, %v), want nil response and JSON error", response, err)
	}
}

func TestAnswerHelpersReturnOnlyMatchingTypes(t *testing.T) {
	response := &SystemOneResponse{Answers: map[string]Answer{
		"choice": ChoiceAnswer{Choice: "a"},
		"score":  ScoreAnswer{Score: 2},
		"noul":   NoulAnswer{Noul: 0.5},
	}}
	if !reflect.DeepEqual(response.Choices(), map[string]ChoiceAnswer{"choice": {Choice: "a"}}) {
		t.Errorf("Choices() = %#v", response.Choices())
	}
	if !reflect.DeepEqual(response.Scores(), map[string]ScoreAnswer{"score": {Score: 2}}) {
		t.Errorf("Scores() = %#v", response.Scores())
	}
	if !reflect.DeepEqual(response.Nouls(), map[string]NoulAnswer{"noul": {Noul: 0.5}}) {
		t.Errorf("Nouls() = %#v", response.Nouls())
	}
}
