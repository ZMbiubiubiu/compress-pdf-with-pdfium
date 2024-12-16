package main

import (
	"compress-pdf/util"
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
	"strings"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/enums"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
)

const (
	PNG  = "png"
	JPEG = "jpeg"
)

const (
	JBIG2DecodeFilter    = "JBIG2Decode"    // JBIG2Decode 是一种高效的二值图像压缩格式，广泛应用于 PDF 文档中，特别是在处理扫描文档和传真图像时
	CCITTFaxDecodeFilter = "CCITTFaxDecode" // CCITTFaxDecode 是一种专为传真图像设计的压缩格式 BitsPerPixel:1
	FlateDecodeFilter    = "FlateDecode"    // PNG
	DCTDecodeFilter      = "DCTDecode"      // JPEG
)

const (
	DPIRecommend = 120 // 水平 DPI 推荐值
)

func CompressImagesInPlace(instance pdfium.Pdfium, inputPath string, quality int, setDPI, highThanDPI float32) error {

	if strings.Contains(inputPath, "compress") {
		return nil
	}

	// 图像信息
	var stat = make(map[string]int)

	// 因为是原地更新，测试阶段，先备份原文件
	copyFilePath := strings.Replace(inputPath, ".pdf", fmt.Sprintf("-compress-%d-%.0fdpi.pdf", quality, setDPI), 1)
	if err := util.CopyFile(inputPath, copyFilePath); err != nil {
		return fmt.Errorf("无法备份原文件: %v", err)
	}

	pdfDoc, err := instance.FPDF_LoadDocument(&requests.FPDF_LoadDocument{
		Path:     &copyFilePath,
		Password: nil,
	})
	if err != nil {
		return fmt.Errorf("无法加载 PDF 文档=%s: %v", inputPath, err)
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

			fmt.Printf("\n\n\n")

			// 获取图片元信息
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

			// 获取图片编码
			filters, err := GetImageObjectFilter(instance, objRes.PageObject)
			if err != nil {
				return err
			}

			// tips:有些pdf中的图片，居然没有filter
			// if len(filters) == 0 {
			// 	fmt.Printf("跳过图片: filter:%s %d-%d\n", strings.Join(filters, ","), i, j)
			// 	continue
			// }

			// 获取图片压缩数据
			dataRawRes, err := instance.FPDFImageObj_GetImageDataRaw(&requests.FPDFImageObj_GetImageDataRaw{
				ImageObject: objRes.PageObject,
			})
			if err != nil {
				return fmt.Errorf("无法获取图片数据: %v", err)
			}

			fmt.Printf("图片元数据: raw len: %d imageMetadataRes:%+v filter:[%s] \n",
				len(dataRawRes.Data), imageMetadataRes.ImageMetadata, strings.Join(filters, ","))

			stat["total-image"]++
			stat[strings.Join(filters, ",")]++
			stat[fmt.Sprintf("color-space-%d", imageMetadataRes.ImageMetadata.Colorspace)]++

			// 图片过小，跳过
			if len(dataRawRes.Data) < 1000 {
				fmt.Printf("图片过小=%d，跳过图片: %d-%d\n", len(dataRawRes.Data), i, j)
				continue
			}

			// if len(dataRawRes.Data) < 1000 || // 图片太小，没必要压缩
			// 	imageMetadataRes.ImageMetadata.BitsPerPixel <= 8 || // bitmap会转为RGB，即BitsPerPixel会变成24
			// 	imageMetadataRes.ImageMetadata.Colorspace == enums.FPDF_COLORSPACE_DEVICECMYK ||
			// 	imageMetadataRes.ImageMetadata.HorizontalDPI/float32(bitmapInfo.Width) > 2 { // 若GetRenderedBitmap得到图片的分辨率已经下降了两倍，不进行处理
			// 	shouldSkipDecode = true
			// }s

			var isSkip bool
			var img image.Image
			var format string

			/*=====================================================step1、提取图片=========================================================*/
			var filter string
			if len(filters) > 0 {
				filter = filters[0]
			}

			switch filter {
			case DCTDecodeFilter, JBIG2DecodeFilter, "":
				img, format, err = GetImageFromBitmap(instance, pdfDoc.Document, requests.Page{
					ByIndex: &requests.PageByIndex{
						Document: pdfDoc.Document,
						Index:    i,
					},
				}, objRes.PageObject)

			case FlateDecodeFilter:
				img, format, err = GetImageFromRenderedBitmap(instance, pdfDoc.Document, requests.Page{
					ByIndex: &requests.PageByIndex{
						Document: pdfDoc.Document,
						Index:    i,
					},
				}, objRes.PageObject)

				// if float32(imageMetadataRes.ImageMetadata.Width)/float32(bitmapInfo.Width) > 2 {
				// 	isSkip = true
				// }
			// case JBIG2DecodeFilter:
			// 	isSkip = true
			case CCITTFaxDecodeFilter:
				isSkip = true
			default:
				isSkip = true
			}

			if err != nil {
				return fmt.Errorf("无法获取图片: %v", err)
			}

			if isSkip {
				fmt.Printf("跳过图片:filter:%s %d-%d\n", strings.Join(filters, ","), i, j)
				continue
			}
			// 记录处理的图片数
			stat["dealed-image"]++

			inputFileName := strings.Split(inputPath, "/")[len(strings.Split(inputPath, "/"))-1]
			filename := fmt.Sprintf("./images-files/%s_%d_%d", inputFileName, i, j)

			/*=====================================================step2、降低图片分辨率=========================================================*/
			if imageMetadataRes.ImageMetadata.HorizontalDPI > highThanDPI {
				img = util.ReduceDPI(img, int(imageMetadataRes.ImageMetadata.Width), imageMetadataRes.ImageMetadata.HorizontalDPI, setDPI)
			}

			/*=====================================================step3、图片压缩=========================================================*/
			switch format {
			case JPEG:
				filename = filename + "." + JPEG
				err = util.ConvertToJPEG(img, filename, quality)
				if err != nil {
					return fmt.Errorf("无法保存图片: %v", err)
				}

				data, err := os.ReadFile(filename)
				if err != nil {
					return fmt.Errorf("无法读取图片: %v", err)
				}
				os.Remove(filename)

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
				log.Printf("JPEG 图像已保存到: %s", filename)

			case PNG:
				filename = filename + "." + PNG
				f, _ := os.Create(filename)
				defer f.Close()
				png.Encode(f, img)
				log.Printf("PNG 图像已保存到: %s", filename)

				bitmapRes, err := CreateBitmapFromImage(instance, img, 0)
				if err != nil {
					return fmt.Errorf("无法创建位图: %v", err)
				}
				_, err = instance.FPDFImageObj_SetBitmap(&requests.FPDFImageObj_SetBitmap{
					ImageObject: objRes.PageObject,
					Bitmap:      bitmapRes.bitmapRef,
					Page: &requests.Page{
						ByIndex: &requests.PageByIndex{
							Document: pdfDoc.Document,
							Index:    i,
						},
					},
					Count: 1,
				})
			}

			if err != nil {
				return fmt.Errorf("无法设置图片: %v", err)
			}
		}

		rectObj, err := instance.FPDFPageObj_CreateNewRect(&requests.FPDFPageObj_CreateNewRect{
			X: 0,
			Y: 0,
			W: 1,
			H: 1,
		})
		if err != nil {
			return fmt.Errorf("无法创建矩形对象: %v", err)
		}

		_, err = instance.FPDFPage_InsertObject(&requests.FPDFPage_InsertObject{
			Page: requests.Page{
				ByIndex: &requests.PageByIndex{
					Document: pdfDoc.Document,
					Index:    i,
				},
			},
			PageObject: rectObj.PageObject,
		})
		if err != nil {
			return fmt.Errorf("无法插入矩形对象: %v", err)
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

	fmt.Printf("压缩后图片信息: %v\n", stat)

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

func GetBitmapInfo(instance pdfium.Pdfium, document references.FPDF_DOCUMENT, page requests.Page, imgObj references.FPDF_PAGEOBJECT, isRendered bool) (*BitmapInfo, error) {
	var bitmap references.FPDF_BITMAP
	if isRendered {
		bitmapRes, err := instance.FPDFImageObj_GetRenderedBitmap(&requests.FPDFImageObj_GetRenderedBitmap{
			ImageObject: imgObj,
			Page:        page,
			Document:    document,
		})
		if err != nil {
			return nil, fmt.Errorf("无法获取图片位图: %v", err)
		}
		bitmap = bitmapRes.Bitmap
	} else {
		bitmapRes, err := instance.FPDFImageObj_GetBitmap(&requests.FPDFImageObj_GetBitmap{
			ImageObject: imgObj,
		})
		if err != nil {
			return nil, fmt.Errorf("无法获取图片位图: %v", err)
		}
		bitmap = bitmapRes.Bitmap
	}

	strideRes, err := instance.FPDFBitmap_GetStride(&requests.FPDFBitmap_GetStride{
		Bitmap: bitmap,
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取图片位图步长: %v", err)
	}

	formatRes, err := instance.FPDFBitmap_GetFormat(&requests.FPDFBitmap_GetFormat{
		Bitmap: bitmap,
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取图片位图格式: %v", err)
	}

	bufferRes, err := instance.FPDFBitmap_GetBuffer(&requests.FPDFBitmap_GetBuffer{
		Bitmap: bitmap,
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取图片位图缓冲区: %v", err)
	}

	bitmapWidthRes, err := instance.FPDFBitmap_GetWidth(&requests.FPDFBitmap_GetWidth{
		Bitmap: bitmap,
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取图片位图宽度: %v", err)
	}

	bitmapHeightRes, err := instance.FPDFBitmap_GetHeight(&requests.FPDFBitmap_GetHeight{
		Bitmap: bitmap,
	})
	if err != nil {
		return nil, fmt.Errorf("无法获取图片位图高度: %v", err)
	}

	bitmapInfo := &BitmapInfo{
		Width:     bitmapWidthRes.Width,
		Height:    bitmapHeightRes.Height,
		Stride:    strideRes.Stride,
		Format:    formatRes.Format,
		Data:      bufferRes.Buffer,
		BitmapRef: bitmap,
	}

	fmt.Printf("bitmap info:%+v\n", bitmapInfo)

	return bitmapInfo, nil
}

func GetImageFromBitmap(instance pdfium.Pdfium, document references.FPDF_DOCUMENT, page requests.Page, imgObj references.FPDF_PAGEOBJECT) (image.Image, string, error) {
	bitmapInfo, err := GetBitmapInfo(instance, document, page, imgObj, false)
	if err != nil {
		return nil, "", fmt.Errorf("无法获取图片位图信息: %v", err)
	}
	defer instance.FPDFBitmap_Destroy(&requests.FPDFBitmap_Destroy{
		Bitmap: bitmapInfo.BitmapRef,
	})
	// fmt.Printf("图片元数据: raw len: %d imageMetadataRes:%+v filter:[%s] bitmap info:%v\n",
	// 	len(dataRawRes.Data), imageMetadataRes.ImageMetadata, strings.Join(filters, ","), bitmapInfo)

	_, img, err := util.RenderImage(bitmapInfo.Data, bitmapInfo.Width, bitmapInfo.Height, bitmapInfo.Stride, int(bitmapInfo.Format))
	if err != nil {
		return nil, "", fmt.Errorf("GetImageFromDCTDecode 无法渲染图片: %v", err)
	}

	return img, JPEG, nil
}

func GetImageFromRenderedBitmap(instance pdfium.Pdfium, document references.FPDF_DOCUMENT, page requests.Page, imgObj references.FPDF_PAGEOBJECT) (image.Image, string, error) {
	bitmapInfo, err := GetBitmapInfo(instance, document, page, imgObj, true)
	if err != nil {
		return nil, "", fmt.Errorf("无法获取图片位图信息: %v", err)
	}
	defer instance.FPDFBitmap_Destroy(&requests.FPDFBitmap_Destroy{
		Bitmap: bitmapInfo.BitmapRef,
	})
	// fmt.Printf("图片元数据: raw len: %d imageMetadataRes:%+v filter:[%s] bitmap info:%v\n",
	// 	len(dataRawRes.Data), imageMetadataRes.ImageMetadata, strings.Join(filters, ","), bitmapInfo)

	isAlphaValid, img, err := util.RenderImage(bitmapInfo.Data, bitmapInfo.Width, bitmapInfo.Height, bitmapInfo.Stride, int(bitmapInfo.Format))
	if err != nil {
		return nil, "", fmt.Errorf("GetImageFromDCTDecode 无法渲染图片: %v", err)
	}

	if isAlphaValid {
		return img, PNG, nil
	}

	return GetImageFromBitmap(instance, document, page, imgObj)

	// return img, JPEG, nil
}
