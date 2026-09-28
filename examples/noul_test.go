package main

import (
	"testing"

	"github.com/yoneyy/typesafe-go"
)

func TestNoul(t *testing.T) {
	client := typesafe.NewClient()
	res, err := client.SystemOne(
		t.Context(),
		typesafe.String("Help! My payouts have been failing for 3 days."),
		typesafe.Questions{
			"is_urgent": typesafe.Noul{
				Instructions: typesafe.String("Does this convey urgency?"),
				Criteria: typesafe.Object{
					"true":  "Explicitly time-sensitive",
					"false": "No urgency expressed",
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

	t.Logf("type: %s", res.Nouls()["is_urgent"].Type)
	t.Logf("noul: %.2f", res.Nouls()["is_urgent"].Noul)
}
