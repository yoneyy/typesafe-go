package main

import (
	"testing"

	"github.com/yoneyy/typesafe-go"
)

func TestChoice(t *testing.T) {
	client := typesafe.NewClient()
	res, err := client.SystemOne(
		t.Context(),
		typesafe.String("Help! My payouts have been failing for 3 days."),
		typesafe.Questions{
			"department": typesafe.Choice{
				Instructions: typesafe.String("Which team should handle this?"),
				Criteria: typesafe.Object{
					"billing":   "Payments, invoicing, refunds",
					"technical": "Bugs, outages, integrations",
					"sales":     "Pricing, upgrades, new accounts",
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

	t.Logf("type: %s", res.Choices()["department"].Type)
	t.Logf("choice: %s", res.Choices()["department"].Choice)
	t.Logf("confidence: %.2f", res.Choices()["department"].Confidence)
	t.Logf("probabilities: %+v", res.Choices()["department"].Probabilities)
}
