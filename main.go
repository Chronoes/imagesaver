package main

// TODO: Run tests for average, dominant and prominent aggregations for various images
// TODO: Intelligent selection of aggregations for images with different backgrounds (based on above test results)

import (
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

func div(base, modulo int) (int, int) {
	return base / modulo, base % modulo
}

func processImage(imageData imageInfo) {
	startTime := time.Now()
	if err := imageData.CompileImage(averageColor); err != nil {
		log.Print(err)
		return
	}
	timeTaken := time.Since(startTime)
	log.Printf("File complete: %s time: %vs", filepath.Base(imageData.Source), timeTaken.Seconds())
}

func processImagesFromJSON(r io.Reader, concurrentProcessingLimit int) {
	decoder := json.NewDecoder(r)
	token, err := decoder.Token()
	if err != nil {
		log.Fatal(err)
	} else if t, ok := token.(json.Delim); !ok || t != '[' {
		log.Fatalln("Expected JSON to be an array")
	}

	var wg sync.WaitGroup
	defer wg.Wait()

	queue := make(chan imageInfo, concurrentProcessingLimit)
	defer close(queue)

	for decoder.More() {
		var imageData imageInfo
		if err = decoder.Decode(&imageData); err != nil {
			log.Fatal(err)
		}
		wg.Add(1)
		queue <- imageData
		go func() {
			processImage(imageData)
			<-queue
			wg.Done()
		}()
		log.Printf("Processing file: %s\n", filepath.Base(imageData.Source))
	}

	_, err = decoder.Token()
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	processImagesFromJSON(os.Stdin, 10)
}
