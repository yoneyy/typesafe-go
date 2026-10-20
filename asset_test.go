package typesafe

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"
)

const boxResponseJSON = `{"model":"jev-1.14.0","usage":{"input_tokens":1,"output_tokens":1},"answers":{"mug":{"type":"box","box":{"top":0.1,"bottom":0.9,"left":0.2,"right":0.8},"confidence":0.9}}}`

func testImageBytes(t *testing.T, format ImageFormat) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	img.Set(1, 1, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	var err error
	if format == ImageFormatPNG {
		err = png.Encode(&buf, img)
	} else {
		err = jpeg.Encode(&buf, img, nil)
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type capturedRequest struct {
	header http.Header
	body   []byte
}

func capturingClient(t *testing.T, requests *[]capturedRequest) *Client {
	t.Helper()
	return clientWithTransport(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		*requests = append(*requests, capturedRequest{header: req.Header.Clone(), body: body})
		return testResponse(http.StatusOK, boxResponseJSON), nil
	})
}

type videoAsset struct{}

func (videoAsset) T() AssetType { return "video" }

func TestImageFormat(t *testing.T) {
	tests := []struct {
		name  string
		image Image
		want  ImageFormat
	}{
		{"detect png", Image{Data: testImageBytes(t, ImageFormatPNG)}, ImageFormatPNG},
		{"detect jpeg", Image{Data: testImageBytes(t, ImageFormatJPEG)}, ImageFormatJPEG},
		{"explicit format wins", Image{Data: testImageBytes(t, ImageFormatPNG), Format: ImageFormatJPG}, ImageFormatJPG},
		{"unknown bytes", Image{Data: []byte("GIF89a")}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.image.format(); got != tt.want {
				t.Fatalf("format() = %q, want %q", got, tt.want)
			}
		})
	}
	if got := (Image{}).T(); got != AssetTypeImage || got.String() != "image" {
		t.Errorf("Image.T() = %q", got)
	}
}

func TestImageFromBase64(t *testing.T) {
	data := testImageBytes(t, ImageFormatPNG)
	img, err := ImageFromBase64(base64.StdEncoding.EncodeToString(data))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(img.Data, data) || img.Format != "" {
		t.Errorf("ImageFromBase64() = %d bytes, format %q", len(img.Data), img.Format)
	}
	if _, err := ImageFromBase64("not base64!"); err == nil || !strings.Contains(err.Error(), "decode base64 image") {
		t.Errorf("ImageFromBase64(invalid) error = %v", err)
	}
}

func TestNewAssetRef(t *testing.T) {
	first, second := NewAssetRef(), NewAssetRef()
	if len(first) < 16 || first == second {
		t.Fatalf("NewAssetRef() = %q, %q; want long, distinct references", first, second)
	}
}

func TestBoxCoordinatesPixels(t *testing.T) {
	tests := []struct {
		name string
		box  BoxCoordinates
		want image.Rectangle
	}{
		{"scaled and rounded", BoxCoordinates{Top: 0.25, Bottom: 0.82, Left: 0.44, Right: 0.84}, image.Rect(440, 125, 840, 410)},
		{"clamped to image", BoxCoordinates{Top: -0.1, Bottom: 1.2, Left: -0.5, Right: 1.5}, image.Rect(0, 0, 1000, 500)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.box.Pixels(1000, 500); got != tt.want {
				t.Fatalf("Pixels() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSystemOneBase64Assets(t *testing.T) {
	var requests []capturedRequest
	client := capturingClient(t, &requests)
	data := testImageBytes(t, ImageFormatPNG)

	response, err := client.SystemOne(context.Background(),
		Object{"photo": "ref-photo"},
		Questions{"mug": Box{Instructions: String("the mug in photo")}},
		WithRequestModel(ModelJevPreview),
		WithAssets(Assets{"ref-photo": Image{Data: data}}),
		WithBase64Assets(),
		WithImages(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if got := response.Boxes()["mug"].Box.Right; got != 0.8 {
		t.Errorf("box right = %v", got)
	}
	if len(requests) != 1 {
		t.Fatalf("request count = %d", len(requests))
	}
	if got := requests[0].header.Get("Content-Type"); got != "application/json" {
		t.Errorf("content type = %q", got)
	}
	if got := requests[0].header.Get(HeaderDangerZoneImages); got != "true" {
		t.Errorf("%s = %q", HeaderDangerZoneImages, got)
	}
	assertJSONEqual(t, `{
		"model":"jev-preview",
		"state":{"photo":"ref-photo"},
		"questions":{"mug":{"type":"box","instructions":"the mug in photo"}},
		"assets":{"ref-photo":{"type":"image","format":"png","data_base64":"`+base64.StdEncoding.EncodeToString(data)+`"}}
	}`, requests[0].body)
}

func TestSystemOneMultipartAssets(t *testing.T) {
	var requests []capturedRequest
	client := capturingClient(t, &requests)
	pngData := testImageBytes(t, ImageFormatPNG)
	jpegData := testImageBytes(t, ImageFormatJPEG)
	rawData := []byte("unrecognized image bytes")

	_, err := client.SystemOne(context.Background(),
		Object{"door": "ref-b", "frames": Array{"ref-a", "ref-c"}},
		Questions{"person": Noul{Instructions: String("Is there a person in door?")}},
		WithAssets(Assets{"ref-b": &Image{Data: pngData}}),
		WithAssets(Assets{"ref-a": Image{Data: jpegData}, "ref-c": Image{Data: rawData}}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 1 {
		t.Fatalf("request count = %d", len(requests))
	}
	mediaType, params, err := mime.ParseMediaType(requests[0].header.Get("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		t.Fatalf("content type = %q, err = %v", requests[0].header.Get("Content-Type"), err)
	}

	reader := multipart.NewReader(bytes.NewReader(requests[0].body), params["boundary"])
	part, err := reader.NextPart()
	if err != nil {
		t.Fatal(err)
	}
	if part.FormName() != "request" || part.FileName() != "" || part.Header.Get("Content-Type") != "application/json" {
		t.Fatalf("first part: name=%q filename=%q content type=%q", part.FormName(), part.FileName(), part.Header.Get("Content-Type"))
	}
	requestJSON, err := io.ReadAll(part)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONEqual(t, `{
		"model":"jev-latest",
		"state":{"door":"ref-b","frames":["ref-a","ref-c"]},
		"questions":{"person":{"type":"noul","instructions":"Is there a person in door?"}},
		"assets":{
			"ref-a":{"type":"image","part":"asset-0"},
			"ref-b":{"type":"image","part":"asset-1"},
			"ref-c":{"type":"image","part":"asset-2"}
		}
	}`, requestJSON)

	wantParts := []struct {
		name        string
		contentType string
		data        []byte
	}{
		{"asset-0", "image/jpeg", jpegData},
		{"asset-1", "image/png", pngData},
		{"asset-2", "application/octet-stream", rawData},
	}
	for _, want := range wantParts {
		part, err := reader.NextPart()
		if err != nil {
			t.Fatalf("part %s: %v", want.name, err)
		}
		data, err := io.ReadAll(part)
		if err != nil {
			t.Fatal(err)
		}
		if part.FormName() != want.name || part.FileName() != want.name ||
			part.Header.Get("Content-Type") != want.contentType || !bytes.Equal(data, want.data) {
			t.Errorf("part: name=%q filename=%q content type=%q, %d bytes; want %s %s, %d bytes",
				part.FormName(), part.FileName(), part.Header.Get("Content-Type"), len(data),
				want.name, want.contentType, len(want.data))
		}
	}
	if _, err := reader.NextPart(); err != io.EOF {
		t.Errorf("extra part after images: %v", err)
	}
}

func TestSystemOneRequestHeaders(t *testing.T) {
	var requests []capturedRequest
	client := capturingClient(t, &requests)

	_, err := client.SystemOne(context.Background(), String("state"),
		Questions{"q": Noul{Instructions: String("Question?")}},
		WithImages(),
		WithLongContext(),
		WithRequestHeader("X-Trace-Id", "abc"),
		WithRequestHeader("Authorization", "Bearer other"),
		WithRequestHeader("Content-Type", "text/plain"),
	)
	if err != nil {
		t.Fatal(err)
	}
	header := requests[0].header
	for key, want := range map[string]string{
		HeaderDangerZoneImages:      "true",
		HeaderDangerZoneLongContext: "true",
		"X-Trace-Id":                "abc",
		"Authorization":             "Bearer test-key",
		"Content-Type":              "application/json",
	} {
		if got := header.Values(key); len(got) != 1 || got[0] != want {
			t.Errorf("header %s = %q, want %q", key, got, want)
		}
	}
}

func TestSystemOneRetryResendsMultipartBody(t *testing.T) {
	var requests []capturedRequest
	client := clientWithTransport(func(req *http.Request) (*http.Response, error) {
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		requests = append(requests, capturedRequest{header: req.Header.Clone(), body: body})
		if len(requests) == 1 {
			return testResponse(http.StatusServiceUnavailable, `{"detail":"busy"}`), nil
		}
		return testResponse(http.StatusOK, boxResponseJSON), nil
	})
	client.maxRetries = 1

	_, err := client.SystemOne(context.Background(), String("ref-photo"),
		Questions{"mug": Box{Instructions: String("the mug")}},
		WithAssets(Assets{"ref-photo": Image{Data: testImageBytes(t, ImageFormatPNG)}}),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 2 {
		t.Fatalf("request count = %d, want 2", len(requests))
	}
	if !bytes.Equal(requests[0].body, requests[1].body) || len(requests[1].body) == 0 ||
		requests[0].header.Get("Content-Type") != requests[1].header.Get("Content-Type") {
		t.Fatalf("retry sent a different body or content type")
	}
}

func TestSystemOneRejectsInvalidAssets(t *testing.T) {
	tests := []struct {
		name string
		opts []SystemOneOption
		want string
	}{
		{"empty reference", []SystemOneOption{WithAssets(Assets{"": Image{Data: []byte("x")}})}, `asset reference "" is empty`},
		{"whitespace reference", []SystemOneOption{WithAssets(Assets{" \t": Image{Data: []byte("x")}})}, `is empty`},
		{"nil asset", []SystemOneOption{WithAssets(Assets{"ref": nil})}, `asset "ref" is nil`},
		{"nil image pointer", []SystemOneOption{WithAssets(Assets{"ref": (*Image)(nil)})}, `asset "ref" is nil`},
		{"empty data", []SystemOneOption{WithAssets(Assets{"ref": Image{}})}, `empty image data`},
		{"unsupported asset", []SystemOneOption{WithAssets(Assets{"ref": videoAsset{}})}, `unsupported asset typesafe.videoAsset`},
		{"unknown base64 format", []SystemOneOption{WithAssets(Assets{"ref": Image{Data: []byte("GIF89a")}}), WithBase64Assets()}, `unknown image format`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := clientWithTransport(func(*http.Request) (*http.Response, error) {
				t.Error("request sent for invalid assets")
				return testResponse(http.StatusOK, boxResponseJSON), nil
			})
			response, err := client.SystemOne(context.Background(), String("ref"),
				Questions{"q": Noul{Instructions: String("Question?")}}, tt.opts...)
			if response != nil || err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("SystemOne() = (%v, %v), want error containing %q", response, err, tt.want)
			}
		})
	}
}
