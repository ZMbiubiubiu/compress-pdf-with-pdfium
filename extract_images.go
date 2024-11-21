package main

import (
	"compress-pdf/util"
	"fmt"
	"strings"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/enums"
	"github.com/klippa-app/go-pdfium/requests"
)

func ExtractImages(instance pdfium.Pdfium, inputPath, outputPath string) error {

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

	// 源文档页面数量
	pageCountRes, err := instance.FPDF_GetPageCount(&requests.FPDF_GetPageCount{
		Document: pdfDoc.Document,
	})
	if err != nil {
		return err
	}

	fmt.Printf("pdf page count: %d\n", pageCountRes.PageCount)

	// 遍历所有页面
	for i := 0; i < pageCountRes.PageCount; i++ {
		fmt.Printf("加载页面:%d\n", i)
		pdfPage, err := instance.FPDF_LoadPage(&requests.FPDF_LoadPage{
			Document: pdfDoc.Document,
			Index:    i,
		})
		if err != nil {
			return fmt.Errorf("无法加载页面: %v", err)
		}

		// 遍历一个页面中的对象
		objectCountRes, err := instance.FPDFPage_CountObjects(&requests.FPDFPage_CountObjects{
			Page: requests.Page{
				ByIndex: &requests.PageByIndex{
					Document: pdfDoc.Document,
					Index:    i,
				},
			},
		})
		if err != nil {
			return fmt.Errorf("无法获取页面对象数量: %v", err)
		}

		for j := 0; j < objectCountRes.Count; j++ {
			objRes, err := instance.FPDFPage_GetObject(&requests.FPDFPage_GetObject{
				Page: requests.Page{
					ByIndex: &requests.PageByIndex{
						Document: pdfDoc.Document,
						Index:    i,
					},
				},
				Index: j,
			})
			if err != nil {
				return fmt.Errorf("无法获取页面对象: %v", err)
			}

			objTypeRes, err := instance.FPDFPageObj_GetType(&requests.FPDFPageObj_GetType{
				PageObject: objRes.PageObject,
			})
			if err != nil {
				return fmt.Errorf("无法获取页面对象类型: %v", err)
			}
			if objTypeRes.Type == enums.FPDF_PAGEOBJ_IMAGE {
				// 	  - FPDFImageObj_GetBitmap
				//    - FPDFBitmap_GetStride
				//    - FPDFBitmap_GetWidth
				//    - FPDFBitmap_GetHeight
				//    - FPDFBitmap_GetFormat

				imageMetadataRes, err := instance.FPDFImageObj_GetImageMetadata(&requests.FPDFImageObj_GetImageMetadata{
					ImageObject: objRes.PageObject,
					Page: requests.Page{
						ByIndex: &requests.PageByIndex{
							Document: pdfDoc.Document,
							Index:    i,
						},
					},
				})
				if err != nil {
					return fmt.Errorf("无法获取图片元数据: %v", err)
				}

				filterCountRes, err := instance.FPDFImageObj_GetImageFilterCount(&requests.FPDFImageObj_GetImageFilterCount{
					ImageObject: objRes.PageObject,
				})
				if err != nil {
					return fmt.Errorf("无法获取图片滤镜数量: %v", err)
				}

				var filters = make([]string, 0, filterCountRes.Count)
				for k := 0; k < filterCountRes.Count; k++ {
					filterRes, err := instance.FPDFImageObj_GetImageFilter(&requests.FPDFImageObj_GetImageFilter{
						ImageObject: objRes.PageObject,
						Index:       k,
					})
					if err != nil {
						return fmt.Errorf("无法获取图片滤镜: %v", err)
					}
					filters = append(filters, filterRes.ImageFilter)
				}

				bitmapInfo, err := GetBitmapInfo(instance, pdfDoc.Document, requests.Page{
					ByIndex: &requests.PageByIndex{
						Document: pdfDoc.Document,
						Index:    i,
					},
				}, objRes.PageObject)
				if err != nil {
					return fmt.Errorf("无法获取图片位图信息: %v", err)
				}

				fmt.Printf("图片元数据: imageMetadataRes:%+v filter:[%s] bitmap info:%s\n",
					imageMetadataRes.ImageMetadata, strings.Join(filters, ","), bitmapInfo)

				filePrefix := fmt.Sprintf("%s/decoded_%d_%d", outputPath, i, j)
				err = util.ConvertToJPEG(bitmapInfo.Width, bitmapInfo.Height, bitmapInfo.Stride, bitmapInfo.Data, filePrefix, 100, int(bitmapInfo.Format))
				if err != nil {
					return fmt.Errorf("无法保存图片: %v", err)
				}
			}
		}

		_, err = instance.FPDF_ClosePage(&requests.FPDF_ClosePage{
			Page: pdfPage.Page,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
