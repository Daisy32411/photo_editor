package processor

import (
	"encoding/json"
	"image"
	"image/color"
	"photo_editor/internal/models"
	"testing"
)

func TestNegative(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))

	img.SetNRGBA(0, 0, color.NRGBA{
		R: 10,
		G: 20,
		B: 30,
		A: 255,
	})

	result := Negative(img)

	got := color.NRGBAModel.Convert(result.At(0, 0)).(color.NRGBA)

	want := color.NRGBA{
		R: 245,
		G: 235,
		B: 225,
		A: 255,
	}

	if got != want {
		t.Errorf("Negative() = %v, want %v", got, want)
	}
}

func TestFlipX(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 2))

	red := color.NRGBA{R: 255, G: 0, B: 0, A: 255}
	blue := color.NRGBA{R: 0, G: 0, B: 255, A: 255}

	img.SetNRGBA(0, 0, red)
	img.SetNRGBA(0, 1, blue)

	result := FlipX(img)

	top := color.NRGBAModel.Convert(result.At(0, 0)).(color.NRGBA)
	bottom := color.NRGBAModel.Convert(result.At(0, 1)).(color.NRGBA)

	if top != blue {
		t.Errorf("top pixel = %v, want %v", top, blue)
	}

	if bottom != red {
		t.Errorf("bottom pixel = %v, want %v", bottom, red)
	}
}

func TestBlur(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 5, 5))

	img.SetNRGBA(2, 2, color.NRGBA{
		R: 255,
		G: 255,
		B: 255,
		A: 255,
	})

	filter := models.Filter{
		Name:       "Blur",
		Parameters: json.RawMessage(`{"sigma": 2.5}`),
	}

	result, err := ApplyFilter(img, filter)
	if err != nil {
		t.Fatalf("ApplyFilter(Blur) failed: %v", err)
	}

	if result == nil {
		t.Fatal("Blur returned nil image")
	}

	if result.Bounds() != img.Bounds() {
		t.Errorf(
			"unexpected bounds: got %v, want %v",
			result.Bounds(),
			img.Bounds(),
		)
	}
}

func TestSharpen(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 5, 5))

	img.SetNRGBA(2, 2, color.NRGBA{
		R: 255,
		G: 255,
		B: 255,
		A: 255,
	})

	filter := models.Filter{
		Name:       "Sharpen",
		Parameters: json.RawMessage(`{"sigma": 1.5}`),
	}

	result, err := ApplyFilter(img, filter)
	if err != nil {
		t.Fatalf("ApplyFilter(Sharpen) failed: %v", err)
	}

	if result == nil {
		t.Fatal("Sharpen returned nil image")
	}

	if result.Bounds() != img.Bounds() {
		t.Errorf(
			"unexpected bounds: got %v, want %v",
			result.Bounds(),
			img.Bounds(),
		)
	}
}

func TestParseSigma(t *testing.T) {
	tests := []struct {
		name       string
		parameters string
		wantSigma  float64
		wantError  bool
	}{
		{
			name:       "valid sigma",
			parameters: `{"sigma": 2.5}`,
			wantSigma:  2.5,
			wantError:  false,
		},
		{
			name:       "zero sigma",
			parameters: `{"sigma": 0}`,
			wantSigma:  0,
			wantError:  true,
		},
		{
			name:       "negative sigma",
			parameters: `{"sigma": -2}`,
			wantSigma:  0,
			wantError:  true,
		},
		{
			name:       "too large sigma",
			parameters: `{"sigma": 21}`,
			wantSigma:  0,
			wantError:  true,
		},
		{
			name:       "invalid type",
			parameters: `{"sigma": "hello"}`,
			wantSigma:  0,
			wantError:  true,
		},
		{
			name:       "missing parameters",
			parameters: "",
			wantSigma:  0,
			wantError:  true,
		},
		{
			name:       "missing sigma",
			parameters: `{}`,
			wantSigma:  0,
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := models.Filter{
				Name:       "Blur",
				Parameters: json.RawMessage(tt.parameters),
			}

			sigma, err := parseSigma(filter)
			if tt.wantError {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
				return
			}

			if sigma != tt.wantSigma {
				t.Errorf("sigma = %v, want %v", sigma, tt.wantSigma)
			}
		})
	}
}

func TestUnknownFilter(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))

	filter := models.Filter{
		Name: "SomeUnknownFilter",
	}

	result, err := ApplyFilter(img, filter)
	if err == nil {
		t.Fatal("expected error for unknown filter")
	}
	if result != nil {
		t.Error("expected nil image for unknown filter")
	}
}
