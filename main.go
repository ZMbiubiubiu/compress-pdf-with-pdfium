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

func AddLogo() {
	logoFile := "/Users/zhangmeng56/Documents/github/go-practice/pdf/compress-pdf-with-pdfium-main/convert.png"
	files := []string{
		"../pdf-files/cbook.pdf",
		// "../pdf_files/nologo.pdf",
	}
	var err error

	for _, file := range files {
		err = PDFAddLogoV1(context.Background(), instance, logoFile, file, strings.Replace(file, ".pdf", "_logo_v1.pdf", 1), 1)
		if err != nil {
			log.Fatalf("PDFAddLogoV1 添加logo失败: %v", err)
		}
		err = PDFAddLogoV2(context.Background(), instance, logoFile, file, strings.Replace(file, ".pdf", "_logo_v2.pdf", 1), 1)
		if err != nil {
			log.Fatalf("PDFAddLogoV2 添加logo失败: %v", err)
		}
	}
}

func CompressPDF() {
	inputPath := "../pdf-files/coding.pdf"

	if err := CompressImagesInPlace(instance, inputPath, 90); err != nil {
		log.Fatalf("压缩 PDF 失败: %v", err)
	}
}

func main() {
	beginTime := time.Now()

	CompressPDF()
	// AddLogo()

	fmt.Printf("PDF 压缩成功，耗时: %dms\n", time.Since(beginTime).Milliseconds())
}
