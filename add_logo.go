package main

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
	"github.com/klippa-app/go-pdfium/structs"
)

type BitmapCreateResponse struct {
	ref    references.FPDF_BITMAP
	width  int
	height int
}

func CreateBitmap(instance pdfium.Pdfium, imgPath string, alpha int) (BitmapCreateResponse, error) {
	var res BitmapCreateResponse
	// 获取水印图片
	watermarkTemp, err := os.Open(imgPath)
	if err != nil {
		return res, err
	}
	defer watermarkTemp.Close()

	watermark, err := png.Decode(watermarkTemp)
	if err != nil {
		return res, err
	}

	// 确保图像为RGBA格式
	var rgbaImg *image.RGBA
	if rgba, ok := watermark.(*image.RGBA); ok {
		rgbaImg = rgba
	} else {
		// 如果图像不是RGBA格式，则进行转换
		rgbaImg = image.NewRGBA(watermark.Bounds())
		draw.Draw(rgbaImg, rgbaImg.Bounds(), watermark, image.Point{}, draw.Src)
	}

	// 限制水印长宽不超过3000
	watermarkWidth, watermarkHeight := 595, 842
	if rgbaImg.Rect.Dx() > 0 && rgbaImg.Rect.Dy() > 0 {
		watermarkWidth = rgbaImg.Rect.Dx()
		watermarkHeight = rgbaImg.Rect.Dy()
	}

	watermarkBitmap, err := instance.FPDFBitmap_Create(&requests.FPDFBitmap_Create{
		Width:  watermarkWidth,
		Height: watermarkHeight,
		Alpha:  alpha,
	})
	if err != nil {
		return res, err
	}

	res.width = watermarkWidth
	res.height = watermarkHeight
	res.ref = watermarkBitmap.Bitmap

	watermarkBuffer, err := instance.FPDFBitmap_GetBuffer(&requests.FPDFBitmap_GetBuffer{
		Bitmap: watermarkBitmap.Bitmap,
	})
	if err != nil {
		return res, err
	}

	watermarkStride, err := instance.FPDFBitmap_GetStride(&requests.FPDFBitmap_GetStride{
		Bitmap: watermarkBitmap.Bitmap,
	})
	if err != nil {
		return res, err
	}

	stride := int(watermarkStride.Stride)
	// 将PNG图像数据复制到FPDF_BITMAP
	// - width, height 表示图像的宽和高
	// - buffer 是一个字节切片，表示FPDF_BITMAP的内存区域
	// - stride 表示每行的字节跨度（可能包括填充）
	for y := 0; y < watermarkHeight; y++ {
		srcStart := y * watermarkWidth * 4
		dstStart := y * stride
		for x := 0; x < watermarkWidth; x++ {
			// 计算源图像和目标缓冲区的索引
			srcIndex := srcStart + x*4
			dstIndex := dstStart + x*4

			// 复制Alpha通道
			watermarkBuffer.Buffer[dstIndex+3] = rgbaImg.Pix[srcIndex+3]

			// 交换红色和蓝色通道，并复制绿色通道
			watermarkBuffer.Buffer[dstIndex] = rgbaImg.Pix[srcIndex+2]   // Blue
			watermarkBuffer.Buffer[dstIndex+1] = rgbaImg.Pix[srcIndex+1] // Green
			watermarkBuffer.Buffer[dstIndex+2] = rgbaImg.Pix[srcIndex]   // Red
		}
	}

	return res, nil
}

func PDFAddLogo(ctx context.Context, instance pdfium.Pdfium, inputPath, watermarkPath, outputPath string, imageScale int) error {

	// 打开一个新的PDF文档
	pdfDoc, err := instance.FPDF_LoadDocument(&requests.FPDF_LoadDocument{
		Path:     &inputPath,
		Password: nil,
	})
	if err != nil {
		return err
	}

	defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{
		Document: pdfDoc.Document,
	})

	// iDocFlag, err := instance.FPDF_GetDocPermissions(&requests.FPDF_GetDocPermissions{
	// 	Document: pdfDoc.Document,
	// })
	// if err != nil {
	// 	return err
	// }
	// // 检查是否有读保护
	// const FPDFPERM_DOC_OPEN = 0x0004 // 这是一个示例值，请根据pdfium的文档查找具体的权限标志
	// if iDocFlag.DocPermissions&FPDFPERM_DOC_OPEN == 0 {
	// 	return errors.New("document is read-protected")
	// }

	pageCount, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{
		Document: pdfDoc.Document,
	})
	if err != nil {
		return err
	}
	fmt.Printf("pageCount: %d\n", pageCount.PageCount)

	watermarkBitmapRes, err := CreateBitmap(instance, watermarkPath, 1)
	if err != nil {
		return err
	}

	for pageIndex := 0; pageIndex < pageCount.PageCount; pageIndex++ {
		// 获取页面
		pdfPage, err := instance.FPDF_LoadPage(&requests.FPDF_LoadPage{
			Document: pdfDoc.Document,
			Index:    pageIndex,
		})
		if err != nil {
			return err
		}

		pageByIndex := requests.PageByIndex{
			Document: pdfDoc.Document,
			Index:    pageIndex,
		}
		filePdfPage := requests.Page{
			ByIndex:     &pageByIndex,
			ByReference: &pdfPage.Page,
		}

		// 获取页宽
		filePageSize, err := instance.FPDF_GetPageSizeByIndex(&requests.FPDF_GetPageSizeByIndex{
			Document: pdfDoc.Document,
			Index:    pageIndex,
		})
		if err != nil {
			return err
		}

		scale := math.Min(filePageSize.Height, filePageSize.Width) / 595

		watermarkImageObj, err := instance.FPDFPageObj_NewImageObj(&requests.FPDFPageObj_NewImageObj{
			Document: pdfDoc.Document,
		})
		if err != nil {
			return err
		}

		// 将图片加载到ImageObject中，ImageObject是Page中的图片对象
		_, err = instance.FPDFImageObj_SetBitmap(&requests.FPDFImageObj_SetBitmap{
			ImageObject: watermarkImageObj.PageObject,
			Bitmap:      watermarkBitmapRes.ref,
		})
		if err != nil {
			return err
		}

		// 调整图片对象的尺寸和位置
		_, err = instance.FPDFImageObj_SetMatrix(&requests.FPDFImageObj_SetMatrix{
			ImageObject: watermarkImageObj.PageObject,
			Transform: structs.FPDF_FS_MATRIX{
				A: float32(scale) * float32(watermarkBitmapRes.width) / float32(6),
				B: 0,
				C: 0,
				D: float32(scale) * float32(watermarkBitmapRes.height) / float32(6),
				E: float32(filePageSize.Width) - float32(scale)*float32(watermarkBitmapRes.width)/float32(6) - float32(21)/float32(354)*float32(watermarkBitmapRes.width)/float32(imageScale),
				F: float32(7) / float32(500) * float32(filePageSize.Height),
			},
		})
		if err != nil {
			return err
		}

		_, err = instance.FPDFPage_InsertObject(&requests.FPDFPage_InsertObject{
			Page:       filePdfPage,
			PageObject: watermarkImageObj.PageObject,
		})
		if err != nil {
			return err
		}

		_, err = instance.FPDFPage_GenerateContent(&requests.FPDFPage_GenerateContent{
			Page: filePdfPage,
		})
		if err != nil {
			return err
		}

		_, err = instance.FPDF_ClosePage(&requests.FPDF_ClosePage{
			Page: pdfPage.Page,
		})
		if err != nil {
			return err
		}
	}

	// 保存为pdf
	_, err = instance.FPDF_SaveAsCopy(&requests.FPDF_SaveAsCopy{
		Document: pdfDoc.Document,
		FilePath: &outputPath,
	})
	if err != nil {
		return err
	}

	return nil
}
