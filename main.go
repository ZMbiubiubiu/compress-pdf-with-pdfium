package main

import (
	"fmt"
	"log"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/single_threaded"
)

// Be sure to close pools/instances when you're done with them.
var pool pdfium.Pool
var instance pdfium.Pdfium

func init() {
	// Init the PDFium library and return the instance to open documents.
	pool = single_threaded.Init(single_threaded.Config{})

	var err error
	instance, err = pool.GetInstance(time.Second * 30)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	beginTime := time.Now()
	inputPath := "../pdf-files/input-no-logo.pdf" // 初始pdf，大小为681KB
	outputPath := "../pdf-files/output10.pdf"     // 输出 PDF 文件路径
	_ = outputPath

	if err := CompressImagesInPlace(instance, inputPath); err != nil {
		log.Fatalf("压缩 PDF 失败: %v", err)
	}

	fmt.Printf("PDF 压缩成功，耗时: %dms\n", time.Since(beginTime).Milliseconds())

	// util.CompareFileSize(inputPath, outputPath)
}
