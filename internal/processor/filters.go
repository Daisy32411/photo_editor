package processor

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"

	"photo_editor/internal/models"

	"github.com/disintegration/imaging"
)

type SigmaParameters struct {
	Sigma float64 `json:"sigma"`
}

// ApplyFilter выбирает фильтр по его названию
func ApplyFilter(img image.Image, filter models.Filter) (image.Image, error) {
	switch filter.Name {
	case "Negative":
		return Negative(img), nil
	case "Blur":
		sigma, err := parseSigma(filter)
		if err != nil {
			return nil, err
		}
		return imaging.Blur(img, sigma), nil
	case "Sharpen":
		sigma, err := parseSigma(filter)
		if err != nil {
			return nil, err
		}
		return imaging.Sharpen(img, sigma), nil
	case "FlipX":
		return FlipX(img), nil
	default:
		return nil, fmt.Errorf("unsupported filter: %s", filter.Name)
	}
}

// Negative создаёт негатив изображения
func Negative(img image.Image) image.Image {
	bounds := img.Bounds()

	res := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			old := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)

			newColor := color.NRGBA{
				R: 255 - old.R,
				G: 255 - old.G,
				B: 255 - old.B,
				A: old.A,
			}

			res.SetNRGBA(x, y, newColor)
		}
	}

	return res
}

// FlipX отражает изображение относительно оси X (сверху вниз)
func FlipX(img image.Image) image.Image {
	bounds := img.Bounds()

	res := image.NewNRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			sourceY := bounds.Min.Y + bounds.Max.Y - 1 - y

			new := color.NRGBAModel.Convert(img.At(x, sourceY)).(color.NRGBA)

			res.SetNRGBA(x, y, new)
		}
	}

	return res
}

// parseSigma извлекает sigma из JSON-параметров фильтра
func parseSigma(filter models.Filter) (float64, error) {
	if len(filter.Parameters) == 0 {
		return 0, fmt.Errorf("filter %s requires parameters", filter.Name)
	}

	var params SigmaParameters

	err := json.Unmarshal(filter.Parameters, &params)
	if err != nil {
		return 0, fmt.Errorf("invalid filter parameters: %w", err)
	}

	if params.Sigma <= 0 {
		return 0, fmt.Errorf("sigma must be greater than 0")
	}

	if params.Sigma > 20 {
		return 0, fmt.Errorf("sigma must not exceed 20")
	}

	return params.Sigma, nil
}
