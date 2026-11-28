package processor

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"strings"
	"testing"

	"photo_editor/internal/models"
)

func TestProcessImagePixelLimit(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))

	var buf bytes.Buffer

	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}

	data := buf.Bytes()

	binary.BigEndian.PutUint32(data[16:20], 8000)
	binary.BigEndian.PutUint32(data[20:24], 8000)

	checksum := crc32.ChecksumIEEE(data[12:29])

	binary.BigEndian.PutUint32(data[29:33], checksum)

	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode PNG config: %v", err)
	}

	if config.Width != 8000 || config.Height != 8000 {
		t.Fatalf(
			"unexpected dimensions: %dx%d",
			config.Width,
			config.Height,
		)
	}

	message := models.ImageTaskMessage{
		Image: base64.StdEncoding.EncodeToString(data),
		Filter: models.Filter{
			Name: "Negative",
		},
	}

	result, err := processImage(message)

	if err == nil {
		t.Fatal("expected pixel limit error")
	}

	if result != "" {
		t.Fatal("expected empty result")
	}

	if !strings.Contains(
		err.Error(),
		"image dimensions exceed maximum allowed pixels",
	) {
		t.Fatalf("unexpected error: %v", err)
	}
}
