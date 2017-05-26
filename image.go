package main

import (
	"fmt"
	"image"
	"image/color"

	"github.com/cenkalti/dominantcolor"
	"github.com/disintegration/imaging"
	"github.com/lucasb-eyer/go-colorful"
)

type imageInfo struct {
	Source, Destination               string
	AspectRatio                       [2]int
	heightMultiplier, widthMultiplier int
}

type aggregateColor func(image.Image) color.Color

func averageColor(img image.Image) color.Color {
	return imaging.Resize(img, 1, 1, imaging.Box).At(0, 0)
}

func dominantColor(img image.Image) color.Color {
	return dominantcolor.Find(img)
}

func rgbToColorful(r, g, b uint32) colorful.Color {
	return colorful.Color{R: float64(r) / 0xffff, G: float64(g) / 0xffff, B: float64(b) / 0xffff}
}

func prominentColor(img image.Image) color.Color {
	hues := new([360]int)
	chromas := new([360]float64)
	luminances := new([360]float64)

	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			h, c, lum := rgbToColorful(r, g, b).Hcl()
			hueIndex := int(h)
			hues[hueIndex]++
			chromas[hueIndex] += c
			luminances[hueIndex] += lum
		}
	}

	maxIndex := 0
	for i, hueCount := range hues {
		if hues[maxIndex] < hueCount {
			maxIndex = i
		}
	}
	maxHue := float64(hues[maxIndex])
	return colorful.Hcl(float64(maxIndex), chromas[maxIndex]/maxHue, luminances[maxIndex]/maxHue)
}

func calculateEdgeLength(length int) int {
	return int(float32(length) * 0.05)
}

func (i *imageInfo) AggregateEdgeColors(baseImg image.Image, aggregate aggregateColor) (firstEdge, secondEdge color.Color) {
	imgBounds := baseImg.Bounds()
	done := make(chan bool)
	if i.widthMultiplier > i.heightMultiplier {
		edgeLength := calculateEdgeLength(imgBounds.Dy())
		go func() {
			firstEdge = aggregate(imaging.CropAnchor(baseImg, imgBounds.Dx(), edgeLength, imaging.TopLeft))
			done <- true
		}()
		secondEdge = aggregate(imaging.CropAnchor(baseImg, imgBounds.Dx(), edgeLength, imaging.BottomLeft))
	} else {
		edgeLength := calculateEdgeLength(imgBounds.Dx())
		go func() {
			firstEdge = aggregate(imaging.CropAnchor(baseImg, edgeLength, imgBounds.Dy(), imaging.TopLeft))
			done <- true
		}()
		secondEdge = aggregate(imaging.CropAnchor(baseImg, edgeLength, imgBounds.Dy(), imaging.TopRight))
	}
	<-done
	return
}

func (i imageInfo) Normalise(img image.Image) *image.NRGBA {
	return imaging.CropCenter(img, i.AspectRatio[0]*i.widthMultiplier, i.AspectRatio[1]*i.heightMultiplier)
}

func (i imageInfo) CreateSplitBackground(normed image.Image, firstEdge, secondEdge color.Color) *image.NRGBA {
	width := i.AspectRatio[0]
	height := i.AspectRatio[1]
	var position image.Point
	if i.widthMultiplier > i.heightMultiplier {
		width *= i.widthMultiplier
		height *= i.widthMultiplier
		position.X = 0
		position.Y = height / 2
	} else {
		width *= i.heightMultiplier
		height *= i.heightMultiplier
		position.X = width / 2
		position.Y = 0
	}
	return imaging.Paste(imaging.New(width, height, firstEdge), imaging.New(width, height, secondEdge), position)
}

func (i imageInfo) CompileImage(aggregate aggregateColor) error {
	img, err := imaging.Open(i.Source)
	if err != nil {
		return fmt.Errorf("Error opening file: %s; %v", i.Source, err)
	}
	imgBounds := img.Bounds()

	var widthOverflow, heightOverflow int
	i.widthMultiplier, widthOverflow = div(imgBounds.Dx(), i.AspectRatio[0])
	i.heightMultiplier, heightOverflow = div(imgBounds.Dy(), i.AspectRatio[1])
	if !(widthOverflow == 0 && heightOverflow == 0 && i.widthMultiplier == i.heightMultiplier) {
		firstEdge, secondEdge := i.AggregateEdgeColors(img, aggregate)
		normalised := i.Normalise(img)
		splitBackground := i.CreateSplitBackground(normalised, firstEdge, secondEdge)
		err = imaging.Save(imaging.PasteCenter(splitBackground, normalised), i.Destination)
	} else {
		err = imaging.Save(img, i.Destination)
	}

	if err != nil {
		return fmt.Errorf("Error saving image to: %s; %v", i.Destination, err)
	}
	return nil
}
