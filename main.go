package main

// TODO: Run tests for average, dominant and prominent aggregations for various images
// TODO: Intelligent selection of aggregations for images with different backgrounds (based on above test results)
// TODO: Separate the main package to make it independent of server, implement CLI

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func dataHandler(rw http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var imgData imageInfo
	err := decoder.Decode(&imgData)
	if err != nil {
		http.Error(rw, "Error decoding JSON", 400)
		return
	}
	log.Printf("File recieved: %s\n", filepath.Base(imgData.Source))
	startTime := time.Now()
	if err := imgData.CompileImage(); err != nil {
		http.Error(rw, err.Error(), 400)
		return
	}
	rw.Write([]byte(fmt.Sprintf("%s\n", imgData.Destination)))
	timeTaken := time.Since(startTime)
	log.Printf("File complete: %s time: %vs", filepath.Base(imgData.Source), timeTaken.Seconds())
}

func main() {
	http.HandleFunc("/", dataHandler)
	http.HandleFunc("/stop", func(rw http.ResponseWriter, req *http.Request) { os.Exit(0) })
	log.Println("Server started")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
