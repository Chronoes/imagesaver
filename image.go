package main

import (
	"fmt"
	"github.com/cenkalti/dominantcolor"
	"github.com/disintegration/imaging"
	"github.com/lucasb-eyer/go-colorful"
	"image"
	"image/color"
)

type imageInfo struct {
	Source, Destination            string
	AspectRatio, AspectMultipliers [2]int
}

type aggregateColor func(image.Image) color.Color

func averageColor(img image.Image) color.Color {
	return imaging.Resize(img, 1, 1, imaging.Box).At(0, 0)
}

func dominantColor(img image.Image) color.Color {
	return dominantcolor.Find(img)
}

func rgbToColorful(r, g, b uint32) colorful.Color {
	return colorful.Color{float64(r) / 0xffff, float64(g) / 0xffff, float64(b) / 0xffff}
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

func calcEdgeLength(length int) int {
	return int(float32(length) * 0.05)
}

func (i *imageInfo) calcMultipliers(width, height int) {
	i.AspectMultipliers[0] = width / i.AspectRatio[0]
	i.AspectMultipliers[1] = height / i.AspectRatio[1]
}

func (i *imageInfo) EdgeColors(baseImg image.Image, aggregate aggregateColor) (edge1, edge2 color.Color, ok bool) {
	imgBounds := baseImg.Bounds()
	WOverflow := imgBounds.Dx() % i.AspectRatio[0]
	HOverflow := imgBounds.Dy() % i.AspectRatio[1]
	i.calcMultipliers(imgBounds.Dx(), imgBounds.Dy())
	ok = false
	if !(WOverflow == 0 && HOverflow == 0 && i.AspectMultipliers[0] == i.AspectMultipliers[1]) {
		done := make(chan bool)
		if i.AspectMultipliers[0] > i.AspectMultipliers[1] {
			edgeLength := calcEdgeLength(imgBounds.Dy())
			go func() {
				edge1 = aggregate(imaging.CropAnchor(baseImg, imgBounds.Dx(), edgeLength, imaging.TopLeft))
				done <- true
			}()
			edge2 = aggregate(imaging.CropAnchor(baseImg, imgBounds.Dx(), edgeLength, imaging.BottomLeft))
		} else {
			edgeLength := calcEdgeLength(imgBounds.Dx())
			go func() {
				edge1 = aggregate(imaging.CropAnchor(baseImg, edgeLength, imgBounds.Dy(), imaging.TopLeft))
				done <- true
			}()
			edge2 = aggregate(imaging.CropAnchor(baseImg, edgeLength, imgBounds.Dy(), imaging.TopRight))
		}
		ok = <-done
	}
	return
}

func (i imageInfo) Normalise(img image.Image) *image.NRGBA {
	return imaging.CropCenter(img, i.AspectRatio[0]*i.AspectMultipliers[0], i.AspectRatio[1]*i.AspectMultipliers[1])
}

func (i imageInfo) FillEdges(normed image.Image, edge1, edge2 color.Color) *image.NRGBA {
	width := i.AspectRatio[0]
	height := i.AspectRatio[1]
	position := new(image.Point)
	if i.AspectMultipliers[0] > i.AspectMultipliers[1] {
		width *= i.AspectMultipliers[0]
		height *= i.AspectMultipliers[0]
		position.X = 0
		position.Y = height / 2
	} else {
		width *= i.AspectMultipliers[1]
		height *= i.AspectMultipliers[1]
		position.X = width / 2
		position.Y = 0
	}
	splitBackground := imaging.Paste(imaging.New(width, height, edge1), imaging.New(width, height, edge2), *position)
	return imaging.PasteCenter(splitBackground, normed)
}

func (i imageInfo) CompileImage(aggregate aggregateColor) error {
	img, err := imaging.Open(i.Source)
	if err != nil {
		return fmt.Errorf("Error opening file: %s; type: %v", i.Source, err)
	}
	if edge1, edge2, ok := i.EdgeColors(img, aggregate); ok {
		normalised := i.FillEdges(i.Normalise(img), edge1, edge2)
		err = imaging.Save(normalised, i.Destination)
	} else {
		err = imaging.Save(img, i.Destination)
	}
	if err != nil {
		return fmt.Errorf("Error saving image to: %s; type: %v", i.Destination, err)
	}
	return nil
}
