package main

import (
	"fmt"
	"os"
	"runtime"
	"github.com/disintegration/imaging"
)

type picture struct {
	currentPath         string
	targetPath          string
	currentRgbHistogram rgbHistogram
	targetRgbHistogram  rgbHistogram
}

func main() {

	//Initial console output
	printInfo()
	//Read parameters from console
	config := collectConfigInformation()
	//Initialize Window from config and start GUI
	initalizeWindow(config)
	window.Main()
	os.Exit(0)
}

func runDeflickering(config configuration) {
	fmt.Println("Starting...")
	runtime.GOMAXPROCS(config.threads)
	pictures := readDirectory(config.sourceDirectory, config.destinationDirectory)
	progress := createProgressBars(len(pictures))
	progress.container.Start()
	//fmt.Printf("%+v\n", pictures)

	//Analyze and create Histograms
	pictures = forEveryPicture(pictures, progress.bars["analyze"], config.threads, func(pic picture) picture {
		var img, err = imaging.Open(pic.currentPath)
		if err != nil {
			fmt.Printf("'%v': %v\n", pic.targetPath, err)
			os.Exit(2)
		}
		pic.currentRgbHistogram = generateRgbHistogramFromImage(img)
		return pic
	})
	//Calculate global or rolling average
	if config.rollingAverage < 1 {
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
			var start = i - config.rollingAverage
			if start < 0 {
				start = 0
			}
			var end = i + config.rollingAverage
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

	pictures = forEveryPicture(pictures, progress.bars["adjust"], config.threads, func(pic picture) picture {
		var img, _ = imaging.Open(pic.currentPath)
		lut := generateRgbLutFromRgbHistograms(pic.currentRgbHistogram, pic.targetRgbHistogram)
		img = applyRgbLutToImage(img, lut)
		imaging.Save(img, pic.targetPath, imaging.JPEGQuality(config.jpegCompression), imaging.PNGCompressionLevel(0))
		return pic
	})
	progress.container.Stop()
	fmt.Println("Finished.")
}
