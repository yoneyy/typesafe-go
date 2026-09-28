package main

import (
	"testing"

	"github.com/yoneyy/typesafe-go"
)

func TestScore(t *testing.T) {
	client := typesafe.NewClient()
	res, err := client.SystemOne(
		t.Context(),
		typesafe.String("Help! My payouts have been failing for 3 days."),
		typesafe.Questions{
			"frustration": typesafe.Score{
				Instructions: typesafe.String("How frustrated is the customer?"),
				Criteria: typesafe.Array{
					"Calm",
					"Frustrated",
					"Very angry",
				},
			},
		},
	)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("res: %+v", res) // Model Usage Answers

	//for _, a := range res.Answers {
	//	t.Logf("answer: %s", a.T().String())
	//}

	t.Logf("type: %s", res.Scores()["frustration"].Type)
	t.Logf("score: %.2f", res.Scores()["frustration"].Score)
	t.Logf("legend: %+v", res.Scores()["frustration"].Legend)
	t.Logf("confidence: %.2f", res.Scores()["frustration"].Confidence)
	t.Logf("probabilities: %+v", res.Scores()["frustration"].Probabilities)
}
