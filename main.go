package main

import (
	"context"
	"fmt"
	"log"
	"strings"
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

func CompressPDF() {
	inputPath := "../pdf-files/cbook1.pdf"

	if err := CompressImagesInPlace(instance, inputPath, 90); err != nil {
		log.Fatalf("压缩 PDF 失败: %v", err)
	}
}

func main() {
	beginTime := time.Now()

	CompressPDF()

	fmt.Printf("PDF 压缩成功，耗时: %dms\n", time.Since(beginTime).Milliseconds())
}
