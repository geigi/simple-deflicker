package main

import (
	"errors"
	"image"
	"image/color"
	"log"

	"github.com/disintegration/imaging"
)

type lut [256]uint8
type rgbLut struct {
	r lut
	g lut
	b lut
}
type histogram [256]uint32
type rgbHistogram struct {
	r histogram
	g histogram
	b histogram
}

func generateRgbHistogramFromImage(input image.Image, startY int, stopY int) rgbHistogram {
	startY, stopY, err := validateStartAndStopY(input, startY, stopY)
	if err != nil {
		log.Fatalln(err)
	}

	var rgbHistogram rgbHistogram
	for y := startY; y < stopY; y++ {
		for x := input.Bounds().Min.X; x < input.Bounds().Max.X; x++ {
			r, g, b, _ := input.At(x, y).RGBA()
			r = r >> 8
			g = g >> 8
			b = b >> 8
			rgbHistogram.r[r]++
			rgbHistogram.g[g]++
			rgbHistogram.b[b]++
		}
	}
	return rgbHistogram
}

func validateStartAndStopY(input image.Image, startY int, stopY int) (int, int, error) {
	if startY < input.Bounds().Min.Y {
		return -1, -1, errors.New("startY is smaller than minimum Y of image")
	}

	if stopY > input.Bounds().Max.Y {
		return -1, -1, errors.New("stopY is greater than maximum Y of image")
	}

	if stopY == -1 {
		stopY = input.Bounds().Max.Y
	}

	if startY > stopY {
		return -1, -1, errors.New("startY is greater than stopY")
	}

	return startY, stopY, nil
}

func convertToCumulativeRgbHistogram(input rgbHistogram) rgbHistogram {
	var targetRgbHistogram rgbHistogram
	targetRgbHistogram.r[0] = input.r[0]
	targetRgbHistogram.g[0] = input.g[0]
	targetRgbHistogram.b[0] = input.b[0]
	for i := 1; i < 256; i++ {
		targetRgbHistogram.r[i] = targetRgbHistogram.r[i-1] + input.r[i]
		targetRgbHistogram.g[i] = targetRgbHistogram.g[i-1] + input.g[i]
		targetRgbHistogram.b[i] = targetRgbHistogram.b[i-1] + input.b[i]
	}
	return targetRgbHistogram
}

func generateRgbLutFromRgbHistograms(current rgbHistogram, target rgbHistogram) rgbLut {
	currentCumulativeRgbHistogram := convertToCumulativeRgbHistogram(current)
	targetCumulativeRgbHistogram := convertToCumulativeRgbHistogram(target)
	var ratio [3]float64
	ratio[0] = float64(currentCumulativeRgbHistogram.r[255]) / float64(targetCumulativeRgbHistogram.r[255])
	ratio[1] = float64(currentCumulativeRgbHistogram.g[255]) / float64(targetCumulativeRgbHistogram.g[255])
	ratio[2] = float64(currentCumulativeRgbHistogram.b[255]) / float64(targetCumulativeRgbHistogram.b[255])
	for i := 0; i < 256; i++ {
		targetCumulativeRgbHistogram.r[i] = uint32(0.5 + float64(targetCumulativeRgbHistogram.r[i])*ratio[0])
		targetCumulativeRgbHistogram.g[i] = uint32(0.5 + float64(targetCumulativeRgbHistogram.g[i])*ratio[1])
		targetCumulativeRgbHistogram.b[i] = uint32(0.5 + float64(targetCumulativeRgbHistogram.b[i])*ratio[2])
	}

	//Generate LUT
	var lut rgbLut
	var p [3]uint8
	for i := 0; i < 256; i++ {
		for targetCumulativeRgbHistogram.r[p[0]] < currentCumulativeRgbHistogram.r[i] {
			p[0]++
		}
		for targetCumulativeRgbHistogram.g[p[1]] < currentCumulativeRgbHistogram.g[i] {
			p[1]++
		}
		for targetCumulativeRgbHistogram.b[p[2]] < currentCumulativeRgbHistogram.b[i] {
			p[2]++
		}
		lut.r[i] = p[0]
		lut.g[i] = p[1]
		lut.b[i] = p[2]
	}
	return lut
}

func applyRgbLutToImage(input image.Image, lut rgbLut) image.Image {
	result := imaging.AdjustFunc(input, func(c color.NRGBA) color.NRGBA {
		c.R = uint8(lut.r[c.R])
		c.G = uint8(lut.g[c.G])
		c.B = uint8(lut.b[c.B])
		return c
	})
	return result
}
