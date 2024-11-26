package main

import (
	"compress-pdf/util"
	"fmt"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/enums"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
)

const (
	JBIG2DecodeFilter = "JBIG2Decode" // JBIG2Decode 是一种高效的二值图像压缩格式，广泛应用于 PDF 文档中，特别是在处理扫描文档和传真图像时
)

func CompressImagesInPlace(instance pdfium.Pdfium, inputPath string, quality int) error {

	// 因为是原地更新，测试阶段，先备份原文件
	copyFilePath := strings.Replace(inputPath, ".pdf", fmt.Sprintf("-compress-%d-%d.pdf", quality, time.Now().Unix()), 1)
	if err := util.CopyFile(inputPath, copyFilePath); err != nil {
		return fmt.Errorf("无法备份原文件: %v", err)
	}

	pdfDoc, err := instance.FPDF_LoadDocument(&requests.FPDF_LoadDocument{
		Path:     &copyFilePath,
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
		fmt.Printf("\n\n--------------------加载页面:%d\n", i)
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

			// 当前只压缩图像
			if objTypeRes.Type != enums.FPDF_PAGEOBJ_IMAGE {
				continue
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

			dataRawRes, err := instance.FPDFImageObj_GetImageDataRaw(&requests.FPDFImageObj_GetImageDataRaw{
				ImageObject: objRes.PageObject,
			})
			if err != nil {
				return fmt.Errorf("无法获取图片数据: %v", err)
			}

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

			filters, err := GetImageObjectFilter(instance, objRes.PageObject)
			if err != nil {
				return err
			}

			var shouldSkip bool
			for _, filter := range filters {
				switch filter {
				case JBIG2DecodeFilter:
					fmt.Printf("JBIG2DecodeFilter\n")
					shouldSkip = true
				}
			}

			fmt.Printf("图片元数据: raw len: %d imageMetadataRes:%+v filter:[%s] bitmap info:%s\n",
				len(dataRawRes.Data), imageMetadataRes.ImageMetadata, strings.Join(filters, ","), bitmapInfo)

			if shouldSkip {
				continue
			}

			isAlphaValid, img, err := util.RenderImage(bitmapInfo.Data, bitmapInfo.Width, bitmapInfo.Height, bitmapInfo.Stride, int(bitmapInfo.Format))
			if err != nil {
				return fmt.Errorf("无法渲染图片: %v", err)
			}
			// isAlphaValid = true

			inputFileName := strings.Split(inputPath, "/")[len(strings.Split(inputPath, "/"))-1]
			filename := fmt.Sprintf("./images-files/%s_%d_%d", inputFileName, i, j)

			if isAlphaValid {
				// 测试用，留痕，保存为png
				func() {
					filename = filename + ".png"
					outFile, _ := os.Create(filename)
					defer outFile.Close()

					// 将图像编码为 png 格式并写入输出文件
					png.Encode(outFile, img)
				}()

				watermarkBitmap, err := CreateBitmapFromImage(instance, img, 1)
				if err != nil {
					return fmt.Errorf("无法创建位图: %v", err)
				}

				instance.FPDFImageObj_SetBitmap(&requests.FPDFImageObj_SetBitmap{
					ImageObject: objRes.PageObject,
					Bitmap:      watermarkBitmap.bitmapRef,
				})
			} else {
				filename = filename + ".jpeg"
				err = util.ConvertToJPEG(img, filename, quality)
				if err != nil {
					return fmt.Errorf("无法保存图片: %v", err)
				}

				data, err := os.ReadFile(filename)
				if err != nil {
					return fmt.Errorf("无法读取图片: %v", err)
				}

				_, err = instance.FPDFImageObj_LoadJpegFileInline(&requests.FPDFImageObj_LoadJpegFileInline{
					ImageObject: objRes.PageObject,
					Page: &requests.Page{
						ByIndex: &requests.PageByIndex{
							Document: pdfDoc.Document,
							Index:    i,
						},
					},
					Count: 1,
					// FilePath: filename,
					FileData: data,
				})
				if err != nil {
					fmt.Printf("FPDFImageObj_LoadJpegFileInline: %v\n", err)
					return fmt.Errorf("无法加载图片: %v", err)
				}
			}
			// os.Remove(filename)

			if _, err = instance.FPDFBitmap_Destroy(&requests.FPDFBitmap_Destroy{
				Bitmap: bitmapInfo.BitmapRef,
			}); err != nil {
				return err
			}

		}

		_, err = instance.FPDFPage_GenerateContent(&requests.FPDFPage_GenerateContent{
			Page: requests.Page{
				ByIndex: &requests.PageByIndex{
					Document: pdfDoc.Document,
					Index:    i,
				},
			},
		})
		if err != nil {
			return fmt.Errorf("无法生成页面内容: %v", err)
		}

		_, err = instance.FPDF_ClosePage(&requests.FPDF_ClosePage{
			Page: pdfPage.Page,
		})
		if err != nil {
			return err
		}
	}

	_, err = instance.FPDF_SaveAsCopy(&requests.FPDF_SaveAsCopy{
		Document: pdfDoc.Document,
		FilePath: &copyFilePath,
		Flags:    requests.SaveFlagNoIncremental,
	})
	if err != nil {
		return fmt.Errorf("无法保存 PDF: %v", err)
	}

	util.CompareFileSize(inputPath, copyFilePath)

	return nil
}

// 获取图片对象信息
func GetImageObjectFilter(instance pdfium.Pdfium, imgObj references.FPDF_PAGEOBJECT) ([]string, error) {

	filterCountRes, err := instance.FPDFImageObj_GetImageFilterCount(&requests.FPDFImageObj_GetImageFilterCount{
		ImageObject: imgObj,
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取图片滤镜数量: %v", err)
	}

	var filters = make([]string, 0, filterCountRes.Count)
	for k := 0; k < filterCountRes.Count; k++ {
		filterRes, err := instance.FPDFImageObj_GetImageFilter(&requests.FPDFImageObj_GetImageFilter{
			ImageObject: imgObj,
			Index:       k,
		})
		if err != nil {
			return nil, fmt.Errorf("无法获取图片滤镜: %v", err)
		}
		filters = append(filters, filterRes.ImageFilter)
	}
	return filters, nil
}

type BitmapInfo struct {
	Width     int
	Height    int
	Stride    int
	Format    enums.FPDF_BITMAP_FORMAT
	Data      []byte
	BitmapRef references.FPDF_BITMAP
}

func (b *BitmapInfo) String() string {
	return fmt.Sprintf("width:%d height:%d stride:%d format:%d data len:%d", b.Width, b.Height, b.Stride, b.Format, len(b.Data))
}

func GetBitmapInfo(instance pdfium.Pdfium, document references.FPDF_DOCUMENT, page requests.Page, imgObj references.FPDF_PAGEOBJECT) (*BitmapInfo, error) {
	bitmapRes, err := instance.FPDFImageObj_GetRenderedBitmap(&requests.FPDFImageObj_GetRenderedBitmap{
		ImageObject: imgObj,
		Page:        page,
		Document:    document,
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取图片位图: %v", err)
	}

	// bitmapRes, err := instance.FPDFImageObj_GetBitmap(&requests.FPDFImageObj_GetBitmap{
	// 	ImageObject: imgObj,
	// 	// Page:        page,
	// 	// Document:    document,
	// })
	if err != nil {
		return nil, fmt.Errorf("无法获取图片位图: %v", err)
	}

	strideRes, err := instance.FPDFBitmap_GetStride(&requests.FPDFBitmap_GetStride{
		Bitmap: bitmapRes.Bitmap,
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取图片位图步长: %v", err)
	}

	formatRes, err := instance.FPDFBitmap_GetFormat(&requests.FPDFBitmap_GetFormat{
		Bitmap: bitmapRes.Bitmap,
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取图片位图格式: %v", err)
	}

	bufferRes, err := instance.FPDFBitmap_GetBuffer(&requests.FPDFBitmap_GetBuffer{
		Bitmap: bitmapRes.Bitmap,
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取图片位图缓冲区: %v", err)
	}

	bitmapWidthRes, err := instance.FPDFBitmap_GetWidth(&requests.FPDFBitmap_GetWidth{
		Bitmap: bitmapRes.Bitmap,
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取图片位图宽度: %v", err)
	}

	bitmapHeightRes, err := instance.FPDFBitmap_GetHeight(&requests.FPDFBitmap_GetHeight{
		Bitmap: bitmapRes.Bitmap,
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取图片位图高度: %v", err)
	}

	return &BitmapInfo{
		Width:     bitmapWidthRes.Width,
		Height:    bitmapHeightRes.Height,
		Stride:    strideRes.Stride,
		Format:    formatRes.Format,
		Data:      bufferRes.Buffer,
		BitmapRef: bitmapRes.Bitmap,
	}, nil
}
