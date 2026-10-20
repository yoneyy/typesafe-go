package typesafe

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
)

var (
	_ Asset = (*Image)(nil)
)

type AssetType string

func (at AssetType) String() string {
	return string(at)
}

const (
	AssetTypeImage AssetType = "image"
)

type Asset interface {
	T() AssetType // image
}

// Assets maps each reference to its asset. Write the reference wherever the
// asset belongs in the state, instructions, or criteria.
type Assets map[string]Asset

type ImageFormat string

func (f ImageFormat) String() string {
	return string(f)
}

const (
	ImageFormatPNG  ImageFormat = "png"
	ImageFormatJPG  ImageFormat = "jpg"
	ImageFormatJPEG ImageFormat = "jpeg"
)

type Image struct {
	Data   []byte      // PNG or JPEG file bytes
	Format ImageFormat // detected from Data when empty
}

func (Image) T() AssetType { return AssetTypeImage }

func ImageFromBase64(s string) (Image, error) {
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return Image{}, fmt.Errorf("typesafe: decode base64 image: %w", err)
	}
	return Image{Data: data}, nil
}

// NewAssetRef returns a random reference that is unlikely to occur in
// ordinary text, as the API matches references anywhere inside strings.
func NewAssetRef() string {
	return rand.Text()
}

func (i Image) format() ImageFormat {
	if i.Format != "" {
		return i.Format
	}
	switch http.DetectContentType(i.Data) {
	case "image/png":
		return ImageFormatPNG
	case "image/jpeg":
		return ImageFormatJPEG
	}
	return ""
}

func (f ImageFormat) contentType() string {
	switch f {
	case ImageFormatPNG:
		return "image/png"
	case ImageFormatJPG, ImageFormatJPEG:
		return "image/jpeg"
	}
	return "application/octet-stream"
}

func assetImage(ref string, asset Asset) (Image, error) {
	if strings.TrimSpace(ref) == "" {
		return Image{}, fmt.Errorf("typesafe: asset reference %q is empty", ref)
	}
	var img Image
	switch a := asset.(type) {
	case Image:
		img = a
	case *Image:
		if a == nil {
			return Image{}, fmt.Errorf("typesafe: asset %q is nil", ref)
		}
		img = *a
	case nil:
		return Image{}, fmt.Errorf("typesafe: asset %q is nil", ref)
	default:
		return Image{}, fmt.Errorf("typesafe: asset %q: unsupported asset %T", ref, asset)
	}
	if len(img.Data) == 0 {
		return Image{}, fmt.Errorf("typesafe: asset %q: empty image data", ref)
	}
	return img, nil
}
