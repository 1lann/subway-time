package main

import (
	"encoding/json"
	"image"
	"image/color"
	"log"
	"time"
)

func imageToDBCommand(img image.Image) json.RawMessage {
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	bmp := make([]int, 0, w*h)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()

			r8 := uint8(r >> 8)
			g8 := uint8(g >> 8)
			b8 := uint8(b >> 8)

			colorInt := int(r8)<<16 | int(g8)<<8 | int(b8) // 0xRRGGBB
			bmp = append(bmp, colorInt)
		}
	}

	data, err := json.Marshal([]any{0, 0, w, h, bmp})
	if err != nil {
		log.Panicf("invalid json payload for imageToDBCommand: %v", err)
	}

	return json.RawMessage(data)
}

// buildAwtrixImagePayload converts an image and text to an Awtrix JSON payload.
func buildAwtrixImagePayload(img image.Image, text string, weatherEffect string) ([]byte, error) {
	payload := map[string]any{
		"draw": []map[string]any{
			{
				"db": imageToDBCommand(img),
			},
		},
		"lifetime": 60,
		"duration": 4,
		"noScroll": true,
	}

	if text != "" {
		payload["text"] = text
		payload["topText"] = true
		payload["textCase"] = 2
		payload["textOffset"] = 12
		payload["center"] = false
	}

	if weatherEffect != "" {
		payload["overlay"] = weatherEffect
	}

	return json.Marshal(payload)
}

func SubwayDataToImage(data *SubwayData, bullet [][]color.Color) image.Image {
	now := time.Now()

	getMins := func(t *TrainData) int {
		diff := t.ArrivalTime.Sub(now)
		if diff < 0 {
			return 0
		}
		return int(diff.Minutes())
	}

	// default values
	m1, m2 := "", ""
	if len(data.NextTrains) > 0 {
		m1 = format2(getMins(data.NextTrains[0]))
	}
	if len(data.NextTrains) > 1 {
		m2 = format2(getMins(data.NextTrains[1]))
	}

	var text string
	if m1 != "" {
		text = m1

		if m2 != "" {
			text += "," + m2
		}
	}

	// Image dimensions
	width := 32
	height := 8

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Draw R logo
	drawSprite(img, bullet, 2, 0)

	// Draw direction arrow on top of R bullet
	arrow := upArrow
	if data.Direction == DirectionSouth {
		arrow = downArrow
	}
	drawSprite(img, arrow, 0, 0)

	// Draw digits
	cx := 8 + 4 // start after logo
	for _, ch := range text {
		sprite := digits[ch]
		drawSprite(img, sprite, cx, 0)
		cx += len(sprite[0]) + 1
	}

	return img
}
