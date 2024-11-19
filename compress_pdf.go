package main

import (
	"compress-pdfium/util"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"strings"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/enums"
	"github.com/klippa-app/go-pdfium/references"
	"github.com/klippa-app/go-pdfium/requests"
)

func CompressImagesInPlace(instance pdfium.Pdfium, inputPath string, quality int) error {

	// 因为是原地更新，测试阶段，先备份原文件
	copyFilePath := strings.Replace(inputPath, ".pdf", fmt.Sprintf("-compress-%d.pdf", time.Now().UnixNano()), 1)
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

			// log.Printf("对象类型：%d\n", objTypeRes.Type)

			if objTypeRes.Type == enums.FPDF_PAGEOBJ_IMAGE {
				// 	  - FPDFImageObj_GetBitmap
				//    - FPDFBitmap_GetStride
				//    - FPDFBitmap_GetWidth
				//    - FPDFBitmap_GetHeight
				//    - FPDFBitmap_GetFormat

				bitmapInfo, err := GetBitmapInfo(instance, objRes.PageObject)
				if err != nil {
					return fmt.Errorf("无法获取图片位图信息: %v", err)
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

				fmt.Printf("图片元数据: imageMetadataRes:%+v filter:[%s] bitmap info:%s\n",
					imageMetadataRes.ImageMetadata, strings.Join(filters, ","), bitmapInfo)

				filePrefix := fmt.Sprintf("./images-files/decoded_%d_%d", i, j)
				err = util.ConvertToJPEG(bitmapInfo.Width, bitmapInfo.Height, bitmapInfo.Stride, bitmapInfo.Data, filePrefix, quality, int(bitmapInfo.Format))
				if err != nil {
					return fmt.Errorf("无法保存图片: %v", err)
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
					// FileData: data,
					FilePath: filePrefix,
				})
				if err != nil {
					fmt.Printf("FPDFImageObj_LoadJpegFileInline: %v\n", err)
					return fmt.Errorf("无法加载图片: %v", err)
				}

				os.Remove(filePrefix)
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
	Width  int
	Height int
	Stride int
	Format enums.FPDF_BITMAP_FORMAT
	Data   []byte
}

func (b *BitmapInfo) String() string {
	return fmt.Sprintf("width:%d height:%d stride:%d format:%d data len:%d", b.Width, b.Height, b.Stride, b.Format, len(b.Data))
}

func GetBitmapInfo(instance pdfium.Pdfium, imgObj references.FPDF_PAGEOBJECT) (*BitmapInfo, error) {
	bitmapRes, err := instance.FPDFImageObj_GetBitmap(&requests.FPDFImageObj_GetBitmap{
		ImageObject: imgObj,
	})
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
		Width:  bitmapWidthRes.Width,
		Height: bitmapHeightRes.Height,
		Stride: strideRes.Stride,
		Format: formatRes.Format,
		Data:   bufferRes.Buffer,
	}, nil
}

// todo 将所有图片变为灰度图
func CompressImagesToGrayInPlace(instance pdfium.Pdfium, inputPath string) error {

	// 因为是原地更新，测试阶段，先备份原文件
	copyFilePath := strings.Replace(inputPath, ".pdf", fmt.Sprintf(".%d.pdf", time.Now().UnixNano()), 1)
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

			// log.Printf("对象类型：%d\n", objTypeRes.Type)

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

				bitmapRes, err := instance.FPDFImageObj_GetBitmap(&requests.FPDFImageObj_GetBitmap{
					ImageObject: objRes.PageObject,
				})
				if err != nil {
					return fmt.Errorf("无法获取图片位图: %v", err)
				}

				formatRes, err := instance.FPDFBitmap_GetFormat(&requests.FPDFBitmap_GetFormat{
					Bitmap: bitmapRes.Bitmap,
				})
				if err != nil {
					return fmt.Errorf("无法获取图片位图格式: %v", err)
				}

				bufferRes, err := instance.FPDFBitmap_GetBuffer(&requests.FPDFBitmap_GetBuffer{
					Bitmap: bitmapRes.Bitmap,
				})
				if err != nil {
					return fmt.Errorf("无法获取图片位图缓冲区: %v", err)
				}

				bitmapWidthRes, err := instance.FPDFBitmap_GetWidth(&requests.FPDFBitmap_GetWidth{
					Bitmap: bitmapRes.Bitmap,
				})
				if err != nil {
					return fmt.Errorf("无法获取图片位图宽度: %v", err)
				}

				bitmapHeightRes, err := instance.FPDFBitmap_GetHeight(&requests.FPDFBitmap_GetHeight{
					Bitmap: bitmapRes.Bitmap,
				})
				if err != nil {
					return fmt.Errorf("无法获取图片位图高度: %v", err)
				}

				// 创建图像文件
				img := image.NewGray(image.Rect(0, 0, bitmapWidthRes.Width, bitmapHeightRes.Height)) // width 和 height 需要根据实际位图大小设置
				copy(img.Pix, bufferRes.Buffer)

				// 保存为 JPEG 文件
				outFileName := fmt.Sprintf("./images-files/bitmap_%d_%d.jpg", i, j)
				outFile, err := os.Create(outFileName)
				if err != nil {
					return fmt.Errorf("无法创建文件: %v", err)
				}
				defer outFile.Close()

				if err := jpeg.Encode(outFile, img, nil); err != nil {
					return fmt.Errorf("无法保存图像: %v", err)
				}

				fmt.Printf("图片元数据: filter:[%s] buffer:%d format:%d %+v\n", strings.Join(filters, ","), len(bufferRes.Buffer), formatRes.Format, imageMetadataRes.ImageMetadata)

				imageDataRawRes, err := instance.FPDFImageObj_GetImageDataRaw(&requests.FPDFImageObj_GetImageDataRaw{
					ImageObject: objRes.PageObject,
				})
				if err != nil {
					return fmt.Errorf("无法获取图片解码数据: %v", err)
				}

				imageDataDecodedRes, err := instance.FPDFImageObj_GetImageDataDecoded(&requests.FPDFImageObj_GetImageDataDecoded{
					ImageObject: objRes.PageObject,
				})
				if err != nil {
					return fmt.Errorf("无法获取图片解码数据: %v", err)
				}

				fmt.Printf("imageDataRawRes: %d imageDataDecodedRes: %d\n", len(imageDataRawRes.Data), len(imageDataDecodedRes.Data))

				filePrefix := fmt.Sprintf("./images-files/decoded_%d_%d", i, j)
				filename, err := util.SaveImageFromData(imageDataDecodedRes.Data, filePrefix)
				if err != nil {
					if strings.Contains(err.Error(), "无法解码图片") {
						fmt.Printf("无法解码图片: %v\n", err)
						continue
					}
					return fmt.Errorf("无法保存图片: %v", err)
				}

				defer os.Remove(filename)

				// bitmapRes, err := instance.FPDFImageObj_GetBitmap(&requests.FPDFImageObj_GetBitmap{
				// 	ImageObject: objRes.PageObject,
				// })
				// if err != nil {
				// 	return fmt.Errorf("无法获取图片位图: %v", err)
				// }
				// fmt.Printf("bitmapRes: %+v\n", bitmapRes.Bitmap)

				// strideRes, err := instance.FPDFBitmap_GetStride(&requests.FPDFBitmap_GetStride{
				// 	Bitmap: bitmapRes.Bitmap,
				// })
				// if err != nil {
				// 	return fmt.Errorf("无法获取图片位图: %v", err)
				// }
				// fmt.Printf("strideRes: %+v\n", strideRes)

				// widthRes, err := instance.FPDFBitmap_GetWidth(&requests.FPDFBitmap_GetWidth{
				// 	Bitmap: bitmapRes.Bitmap,
				// })
				// if err != nil {
				// 	return err
				// }
				// fmt.Printf("widthRes: %+v\n", widthRes)

				// heightRes, err := instance.FPDFBitmap_GetHeight(&requests.FPDFBitmap_GetHeight{
				// 	Bitmap: bitmapRes.Bitmap,
				// })
				// if err != nil {
				// 	return err
				// }
				// fmt.Printf("heightRes: %+v\n", heightRes)

				// formatRes, err := instance.FPDFBitmap_GetFormat(&requests.FPDFBitmap_GetFormat{
				// 	Bitmap: bitmapRes.Bitmap,
				// })
				// if err != nil {
				// 	return fmt.Errorf("无法获取图片格式: %v", err)
				// }
				// fmt.Printf("formatRes: %+v\n", formatRes)

				// data, err := os.ReadFile(filename)
				// if err != nil {
				// 	return err
				// }
				// _ = data

				_, err = instance.FPDFImageObj_LoadJpegFile(&requests.FPDFImageObj_LoadJpegFile{
					ImageObject: objRes.PageObject,
					Page: &requests.Page{
						ByIndex: &requests.PageByIndex{
							Document: pdfDoc.Document,
							Index:    i,
						},
					},
					Count: 1,
					// FileData: data,
					FilePath: outFileName,
				})
				if err != nil {
					fmt.Printf("FPDFImageObj_LoadJpegFileInline: %v\n", err)
					return fmt.Errorf("无法加载图片: %v", err)
				}

				// _, err = instance.FPDFPage_InsertObject(&requests.FPDFPage_InsertObject{
				// 	Page:       requests.Page{ByReference: &pdfPage.Page},
				// 	PageObject: objRes.PageObject,
				// })
				// if err != nil {
				// 	return fmt.Errorf("无法插入图片对象: %v", err)
				// }
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
