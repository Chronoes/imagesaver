package main

/**
 * TODO: Dominant colours of the edge instead of average
 * Gradient colour from the content to new image edge
 */

import (
	"encoding/json"
	"fmt"
	"github.com/disintegration/imaging"
	"image"
	"image/color"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type imageMod struct {
	Source, Destination            string
	AspectRatio, AspectMultipliers [2]int
}

func (i *imageMod) CalcMultipliers(width, height int) {
	i.AspectMultipliers[0] = width / i.AspectRatio[0]
	i.AspectMultipliers[1] = height / i.AspectRatio[1]
}

func (i *imageMod) EdgeColors(baseImg image.Image) (edge1, edge2 color.Color, ok bool) {
	imgBounds := baseImg.Bounds()
	WOverflow := imgBounds.Dx() % i.AspectRatio[0]
	HOverflow := imgBounds.Dy() % i.AspectRatio[1]
	i.CalcMultipliers(imgBounds.Dx(), imgBounds.Dy())
	ok = false
	done := make(chan bool)
	if !(WOverflow == 0 && HOverflow == 0 && i.AspectMultipliers[0] == i.AspectMultipliers[1]) {
		if i.AspectMultipliers[0] > i.AspectMultipliers[1] {
			edgelen := calcEdgeLength(imgBounds.Dy())
			go func() {
				edge1 = averageColor(imaging.CropAnchor(baseImg, imgBounds.Dx(), edgelen, imaging.TopLeft))
				done <- true
			}()
			edge2 = averageColor(imaging.CropAnchor(baseImg, imgBounds.Dx(), edgelen, imaging.BottomLeft))
		} else {
			edgelen := calcEdgeLength(imgBounds.Dx())
			go func() {
				edge1 = averageColor(imaging.CropAnchor(baseImg, edgelen, imgBounds.Dy(), imaging.TopLeft))
				done <- true
			}()
			edge2 = averageColor(imaging.CropAnchor(baseImg, edgelen, imgBounds.Dy(), imaging.TopRight))
		}
		ok = <-done
	}
	return
}

func (i imageMod) Normalise(img image.Image) *image.NRGBA {
	return imaging.CropCenter(img, i.AspectRatio[0]*i.AspectMultipliers[0], i.AspectRatio[1]*i.AspectMultipliers[1])
}

func (i imageMod) FillEdges(normed image.Image, edge1, edge2 color.Color) *image.NRGBA {
	width := i.AspectRatio[0]
	height := i.AspectRatio[1]
	pos := new(image.Point)
	if i.AspectMultipliers[0] > i.AspectMultipliers[1] {
		width *= i.AspectMultipliers[0]
		height *= i.AspectMultipliers[0]
		pos.X = 0
		pos.Y = height / 2
	} else {
		width *= i.AspectMultipliers[1]
		height *= i.AspectMultipliers[1]
		pos.X = width / 2
		pos.Y = 0
	}
	splitBg := imaging.Paste(imaging.New(width, height, edge1), imaging.New(width, height, edge2), *pos)
	return imaging.PasteCenter(splitBg, normed)
}

func (i imageMod) CompileImage() (string, int, error) {
	img, err := imaging.Open(i.Source)
	if err != nil {
		return fmt.Sprintf("Error opening file: %s", i.Source), 400, err
	}
	destination := i.Destination + filepath.Base(i.Source)
	if edge1, edge2, ok := i.EdgeColors(img); ok {
		normImg := i.FillEdges(i.Normalise(img), edge1, edge2)
		err = imaging.Save(normImg, destination)
	} else {
		err = imaging.Save(img, destination)
	}
	if err != nil {
		return fmt.Sprintf("Error saving image to: %s", destination), 400, err
	}
	return destination, 200, nil
}

func averageColor(img image.Image) color.Color {
	return imaging.Resize(img, 1, 1, imaging.Box).At(0, 0)
}

func calcEdgeLength(length int) int {
	return int(float32(length) * 0.05)
}

func dataHandler(rw http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var imgData imageMod
	err := decoder.Decode(&imgData)
	log.Printf("File recieved: %s\n", filepath.Base(imgData.Source))
	if err != nil {
		http.Error(rw, "Error decoding JSON", 400)
		return
	}
	message, code, err := imgData.CompileImage()
	if err != nil {
		http.Error(rw, message+" , type: "+(err.Error()), code)
		return
	}
	rw.Write([]byte(fmt.Sprintf("%s\n", message)))
	log.Printf("File complete: %s\n", filepath.Base(imgData.Source))
}

func main() {
	http.HandleFunc("/", dataHandler)
	http.HandleFunc("/stop", func(rw http.ResponseWriter, req *http.Request) { os.Exit(0) })
	log.Println("Server started")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
