package main

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/skratchdot/open-golang/open"

	"github.com/disintegration/imaging"
	"github.com/gosuri/uiprogress"
)

type picture struct {
	currentPath         string
	targetPath          string
	currentRgbHistogram rgbHistogram
	targetRgbHistogram  rgbHistogram
}

func main() {
	printInfo()

	config := collectConfigInformation()

	makeDirectoryIfNotExists(config.destinationDirectory)

	//Set number of CPU cores to use
	runtime.GOMAXPROCS(config.threads)

	pictures := readDirectory(config.sourceDirectory, config.destinationDirectory)
	runDeflickering(pictures, config)
	open.Start(config.destinationDirectory)
	fmt.Println("Finished. This window will close itself in 5 seconds")
	time.Sleep(time.Second * 5)
	os.Exit(0)
}

func runDeflickering(pictures []picture, config configuration) {
	uiprogress.Start() // start rendering
	progressBars := createProgressBars(len(pictures))

	//Analyze and create Histograms
	pictures = forEveryPicture(pictures, progressBars.analyze, config.threads, func(pic picture) picture {
		var img, err = imaging.Open(pic.currentPath)
		if err != nil {
			fmt.Printf("'%v': %v\n", pic.targetPath, err)
			os.Exit(2)
		}
		pic.currentRgbHistogram = generateRgbHistogramFromImage(img, config.startY, config.stopY)
		return pic
	})

	//Calculate global or rolling average
	if config.rollingaverage < 1 {
		var averageRgbHistogram rgbHistogram
		for i := range pictures {
			for j := 0; j < 256; j++ {
				averageRgbHistogram.r[j] += pictures[i].currentRgbHistogram.r[j]
				averageRgbHistogram.g[j] += pictures[i].currentRgbHistogram.g[j]
				averageRgbHistogram.b[j] += pictures[i].currentRgbHistogram.b[j]
			}
		}
		for i := 0; i < 256; i++ {
			averageRgbHistogram.r[i] /= uint32(len(pictures))
			averageRgbHistogram.g[i] /= uint32(len(pictures))
			averageRgbHistogram.b[i] /= uint32(len(pictures))
		}
		for i := range pictures {
			pictures[i].targetRgbHistogram = averageRgbHistogram
		}
	} else {
		for i := range pictures {
			var averageRgbHistogram rgbHistogram
			var start = i - config.rollingaverage
			if start < 0 {
				start = 0
			}
			var end = i + config.rollingaverage
			if end > len(pictures)-1 {
				end = len(pictures) - 1
			}
			for i := start; i <= end; i++ {
				for j := 0; j < 256; j++ {
					averageRgbHistogram.r[j] += pictures[i].currentRgbHistogram.r[j]
					averageRgbHistogram.g[j] += pictures[i].currentRgbHistogram.g[j]
					averageRgbHistogram.b[j] += pictures[i].currentRgbHistogram.b[j]
				}
			}
			for i := 0; i < 256; i++ {
				averageRgbHistogram.r[i] /= uint32(end - start + 1)
				averageRgbHistogram.g[i] /= uint32(end - start + 1)
				averageRgbHistogram.b[i] /= uint32(end - start + 1)
			}
			pictures[i].targetRgbHistogram = averageRgbHistogram
		}
	}

	pictures = forEveryPicture(pictures, progressBars.adjust, config.threads, func(pic picture) picture {
		var img, _ = imaging.Open(pic.currentPath)
		lut := generateRgbLutFromRgbHistograms(pic.currentRgbHistogram, pic.targetRgbHistogram)
		img = applyRgbLutToImage(img, lut)
		imaging.Save(img, pic.targetPath, imaging.JPEGQuality(config.jpegcompression), imaging.PNGCompressionLevel(0))
		return pic
	})
	uiprogress.Stop()
}
