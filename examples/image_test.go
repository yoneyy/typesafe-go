package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"

	"github.com/yoneyy/typesafe-go"
)

// redRectanglePNG draws a 400×400 white image with a red rectangle at
// x 80–280, y 120–260.
func redRectanglePNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 400, 400))
	draw.Draw(img, img.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(80, 120, 280, 260), image.NewUniform(color.RGBA{R: 255, A: 255}), image.Point{}, draw.Src)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestImageMultipart(t *testing.T) {
	client := typesafe.NewClient()
	ref := typesafe.NewAssetRef()
	res, err := client.SystemOne(
		t.Context(),
		typesafe.Object{"photo": ref},
		typesafe.Questions{
			"has_rectangle": typesafe.Noul{
				Instructions: typesafe.String("Is there a rectangle in `photo`?"),
			},
			"color": typesafe.Choice{
				Instructions: typesafe.String("What color is the shape in `photo`?"),
				Criteria: typesafe.Object{
					"red":   "Red",
					"green": "Green",
					"blue":  "Blue",
				},
			},
		},
		typesafe.WithAssets(typesafe.Assets{ref: typesafe.Image{Data: redRectanglePNG(t)}}),
		typesafe.WithImages(),
	)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("noul: %.2f", res.Nouls()["has_rectangle"].Noul)
	t.Logf("choice: %s", res.Choices()["color"].Choice)
}

func TestImageBase64(t *testing.T) {
	client := typesafe.NewClient()
	ref := typesafe.NewAssetRef()
	res, err := client.SystemOne(
		t.Context(),
		typesafe.Object{"photo": ref},
		typesafe.Questions{
			"has_rectangle": typesafe.Noul{
				Instructions: typesafe.String("Is there a rectangle in `photo`?"),
			},
		},
		typesafe.WithAssets(typesafe.Assets{ref: typesafe.Image{Data: redRectanglePNG(t), Format: typesafe.ImageFormatPNG}}),
		typesafe.WithBase64Assets(),
		typesafe.WithImages(),
	)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("noul: %.2f", res.Nouls()["has_rectangle"].Noul)
}

func TestImageInInstructions(t *testing.T) {
	client := typesafe.NewClient()
	ref := typesafe.NewAssetRef()
	res, err := client.SystemOne(
		t.Context(),
		typesafe.String("A user uploaded a picture for review."),
		typesafe.Questions{
			"is_red": typesafe.Noul{
				// The reference works inside a longer string.
				Instructions: typesafe.String("Does this picture show a red shape: " + ref + "?"),
			},
		},
		typesafe.WithAssets(typesafe.Assets{ref: typesafe.Image{Data: redRectanglePNG(t)}}),
		typesafe.WithImages(),
	)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("noul: %.2f", res.Nouls()["is_red"].Noul)
}
