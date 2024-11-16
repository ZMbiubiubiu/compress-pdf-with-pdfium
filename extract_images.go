package main

import (
	"compress-pdfium/util"
	"fmt"
	"log"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/enums"
	"github.com/klippa-app/go-pdfium/requests"
)

func ExtractImages(instance pdfium.Pdfium, inputPath string, outputPath string) error {
	pdfDoc, err := instance.FPDF_LoadDocument(&requests.FPDF_LoadDocument{
		Path:     &inputPath,
		Password: nil,
	})
	if err != nil {
		return fmt.Errorf("无法加载 PDF 文档: %v", err)
	}
	defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{
		Document: pdfDoc.Document,
	})

	// 源文档图片大小
	pageCountRes, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{
		Document: pdfDoc.Document,
	})
	if err != nil {
		return err
	}

	for i := 0; i < pageCountRes.PageCount; i++ {
		log.Println("加载页面:", i)
		pdfPage, err := instance.FPDF_LoadPage(&requests.FPDF_LoadPage{
			Document: pdfDoc.Document,
			Index:    i,
		})
		if err != nil {
			return err
		}

		pageSizeRes, err := instance.FPDF_GetPageSizeByIndex(&requests.FPDF_GetPageSizeByIndex{
			Document: pdfDoc.Document,
			Index:    i,
		})
		if err != nil {
			return err
		}

		log.Printf("页面宽度：%f, 页面高度：%f\n", pageSizeRes.Width, pageSizeRes.Height)

		// 遍历页面中的对象
		objectCountRes, err := instance.FPDFPage_CountObjects(&requests.FPDFPage_CountObjects{
			Page: requests.Page{ByReference: &pdfPage.Page},
		})
		if err != nil {
			return err
		}

		for j := 0; j < objectCountRes.Count; j++ {
			objRes, err := instance.FPDFPage_GetObject(&requests.FPDFPage_GetObject{
				Page:  requests.Page{ByReference: &pdfPage.Page},
				Index: j,
			})
			if err != nil {
				return err
			}

			objTypeRes, err := instance.FPDFPageObj_GetType(&requests.FPDFPageObj_GetType{
				PageObject: objRes.PageObject,
			})
			if err != nil {
				return err
			}

			// log.Printf("对象类型：%d\n", objTypeRes.Type)

			if objTypeRes.Type == enums.FPDF_PAGEOBJ_IMAGE {
				imageDataDecodedRes, err := instance.FPDFImageObj_GetImageDataDecoded(&requests.FPDFImageObj_GetImageDataDecoded{
					ImageObject: objRes.PageObject,
				})
				if err != nil {
					return err
				}

				log.Printf("解码后的图片大小：%d\n", len(imageDataDecodedRes.Data))

				// 保存压缩后的图片
				_, err = util.SaveImageFromData(imageDataDecodedRes.Data, fmt.Sprintf("./images/image_decoded_%d_%d", i, j))
				if err != nil {
					return err
				}

				imageMetadataRes, err := instance.FPDFImageObj_GetImageMetadata(&requests.FPDFImageObj_GetImageMetadata{
					ImageObject: objRes.PageObject,
					Page:        requests.Page{ByReference: &pdfPage.Page},
				})
				if err != nil {
					return err
				}

				log.Printf("图片元数据：%+v\n", imageMetadataRes.ImageMetadata)
			}
		}

	}

	return nil
}
