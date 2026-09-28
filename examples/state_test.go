package main

import (
	"testing"

	"github.com/yoneyy/typesafe-go"
)

func TestStateString(t *testing.T) {
	client := typesafe.NewClient()
	state := typesafe.String("My card was charged twice.")
	res, err := client.SystemOne(
		t.Context(),
		state,
		typesafe.Questions{
			"duplicate_charge": typesafe.Noul{
				Instructions: typesafe.String("Does this message describe a duplicate charge?"),
			},
		},
	)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("noul: %.2f", res.Nouls()["duplicate_charge"].Noul)
}

func TestStateArray(t *testing.T) {
	client := typesafe.NewClient()
	state := typesafe.Array{
		"Hi",
		"My customer number is TS1337.",
		"My card was charged twice.",
	}
	res, err := client.SystemOne(
		t.Context(),
		state,
		typesafe.Questions{
			"duplicate_charge": typesafe.Noul{
				Instructions: typesafe.String("Does this conversation describe a duplicate charge?"),
			},
		},
	)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("noul: %.2f", res.Nouls()["duplicate_charge"].Noul)
}

func TestStateObject(t *testing.T) {
	client := typesafe.NewClient()
	state := typesafe.Object{
		"message":  "My card was charged twice.",
		"order_id": "A-104",
	}
	res, err := client.SystemOne(
		t.Context(),
		state,
		typesafe.Questions{
			"duplicate_charge": typesafe.Noul{
				Instructions: typesafe.String("Does the message describe a duplicate charge?"),
			},
		},
	)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("noul: %.2f", res.Nouls()["duplicate_charge"].Noul)
}

func TestStateComplexObject(t *testing.T) {
	client := typesafe.NewClient()
	state := typesafe.Object{
		"ticket": typesafe.Object{
			"subject": "Duplicate charge",
			"messages": typesafe.Array{
				typesafe.Object{
					"from": "customer",
					"text": "I was charged twice for order A-104. Please refund the duplicate.",
				},
				typesafe.Object{"from": "support", "text": "We are checking the charges."},
			},
		},
		"order": typesafe.Object{
			"id": "A-104",
			"charges": typesafe.Array{
				typesafe.Object{"amount_usd": 49, "status": "captured"},
				typesafe.Object{"amount_usd": 49, "status": "captured"},
			},
		},
		"refund_policy": "Duplicate charges are eligible for a refund.",
	}
	res, err := client.SystemOne(
		t.Context(),
		state,
		typesafe.Questions{
			"refund_eligible": typesafe.Noul{
				Instructions: typesafe.String("Does the refund policy support refunding this duplicate charge?"),
			},
		},
	)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("noul: %.2f", res.Nouls()["refund_eligible"].Noul)
}
