package main

import (
	"testing"

	"github.com/yoneyy/typesafe-go"
)

func TestBox(t *testing.T) {
	client := typesafe.NewClient()
	ref := typesafe.NewAssetRef()
	res, err := client.SystemOne(
		t.Context(),
		typesafe.Object{"photo": ref},
		typesafe.Questions{
			// A Box always returns a box, so ask whether the target is there too.
			"rectangle_visible": typesafe.Noul{
				Instructions: typesafe.String("Is there a red rectangle in `photo`?"),
			},
			"rectangle": typesafe.Box{
				Instructions: typesafe.String("the red rectangle in `photo`"),
			},
		},
		typesafe.WithRequestModel(typesafe.ModelJevPreview), // Box questions need Jev 1.14
		typesafe.WithAssets(typesafe.Assets{ref: typesafe.Image{Data: redRectanglePNG(t)}}),
		typesafe.WithImages(),
	)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("model: %s", res.Model)
	t.Logf("visible: %.2f", res.Nouls()["rectangle_visible"].Noul)

	box := res.Boxes()["rectangle"]
	t.Logf("box: %+v", box.Box) // expected near top 0.30, bottom 0.65, left 0.20, right 0.70
	if box.Confidence != nil {
		t.Logf("confidence: %.2f", *box.Confidence)
	}
	t.Logf("pixels: %v", box.Box.Pixels(400, 400)) // expected near (80,120)-(280,260)
}
