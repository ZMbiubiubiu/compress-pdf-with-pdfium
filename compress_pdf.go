package main

import (
	"compress-pdfium/util"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/klippa-app/go-pdfium"
	"github.com/klippa-app/go-pdfium/enums"
	"github.com/klippa-app/go-pdfium/requests"
)

// 新增函数：提取并压缩 PDF 中的图片
func CompressImagesByRebuild(instance pdfium.Pdfium, inputPath string, outputPath string) error {
	pdfDoc, err := instance.FPDF_LoadDocument(&requests.FPDF_LoadDocument{
		Path:     &inputPath,
		Password: nil,
	})
	if err != nil {
		return fmt.Errorf("无法加载 PDF 文档: %v", err)
	}
	// defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{
	// 	Document: pdfDoc.Document,
	// })

	// 创建新的 PDF 文档
	newPdf, err := instance.FPDF_CreateNewDocument(&requests.FPDF_CreateNewDocument{})
	if err != nil {
		return fmt.Errorf("无法创建新的 PDF 文档: %v", err)
	}
	defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{
		Document: newPdf.Document,
	})

	// 源文档页面数量
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

		// 获取页面大小
		pageSizeRes, err := instance.FPDF_GetPageSizeByIndex(&requests.FPDF_GetPageSizeByIndex{
			Document: pdfDoc.Document,
			Index:    i,
		})
		if err != nil {
			return err
		}

		_, err = instance.FPDFPage_New(&requests.FPDFPage_New{
			Document: newPdf.Document,
			Width:    pageSizeRes.Width,
			Height:   pageSizeRes.Height,
		})
		if err != nil {
			return err
		}

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

			if objTypeRes.Type == enums.FPDF_PAGEOBJ_IMAGE {
				// 处理图片对象
				imageDataDecodedRes, err := instance.FPDFImageObj_GetImageDataDecoded(&requests.FPDFImageObj_GetImageDataDecoded{
					ImageObject: objRes.PageObject,
				})
				if err != nil {
					return err
				}

				log.Printf("解码后的图片大小：%d\n", len(imageDataDecodedRes.Data))

				// 将提取出的图片保持原样插入到新文档中
				_, err = instance.FPDFPage_InsertObject(&requests.FPDFPage_InsertObject{
					Page: requests.Page{
						// ByReference: &newPage.Page,
						ByIndex: &requests.PageByIndex{
							Document: newPdf.Document,
							Index:    i,
						},
					},
					PageObject: objRes.PageObject,
				})
				if err != nil {
					return err
				}
			} else {
				// 将非图片对象原封不动地插入到新页面
				if _, err := instance.FPDFPage_InsertObject(&requests.FPDFPage_InsertObject{
					Page: requests.Page{
						// ByReference: &newPage.Page,
						ByIndex: &requests.PageByIndex{
							Document: newPdf.Document,
							Index:    i,
						},
					},
					PageObject: objRes.PageObject,
				}); err != nil {
					return err
				}
			}

			// 生成新页面的内容
			if _, err := instance.FPDFPage_GenerateContent(&requests.FPDFPage_GenerateContent{
				Page: requests.Page{
					// ByReference: &newPage.Page,
					ByIndex: &requests.PageByIndex{
						Document: newPdf.Document,
						Index:    i,
					},
				},
			}); err != nil {
				return err
			}

		}

	}

	// 保存新的 PDF 文档
	if _, err := instance.FPDF_SaveAsCopy(&requests.FPDF_SaveAsCopy{
		Document: newPdf.Document,
		FilePath: &outputPath,
	}); err != nil {
		return fmt.Errorf("无法保存新的 PDF 文档: %v", err)
	}

	return nil
}

func CompressImagesInPlace(instance pdfium.Pdfium, inputPath string) error {

	// 因为是原地更新，测试阶段，先备份原文件
	outputPath := strings.Replace(inputPath, ".pdf", fmt.Sprintf(".%d.backup.pdf", time.Now().UnixNano()), 1)
	if err := util.CopyFile(inputPath, outputPath); err != nil {
		return fmt.Errorf("无法备份原文件: %v", err)
	}

	pdfDoc, err := instance.FPDF_LoadDocument(&requests.FPDF_LoadDocument{
		Path:     &outputPath,
		Password: nil,
	})
	if err != nil {
		return fmt.Errorf("无法加载 PDF 文档: %v", err)
	}
	defer instance.FPDF_CloseDocument(&requests.FPDF_CloseDocument{
		Document: pdfDoc.Document,
	})

	// 直接生成备份
	// _, err = instance.FPDF_SaveAsCopy(&requests.FPDF_SaveAsCopy{
	// 	Document: pdfDoc.Document,
	// 	FilePath: &outputPath,
	// })
	// if err != nil {
	// 	return fmt.Errorf("无法保存 PDF: %v", err)
	// }

	// util.CompareFileSize(inputPath, outputPath)
	// return nil

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
			Page: requests.Page{ByReference: &pdfPage.Page},
		})
		if err != nil {
			return fmt.Errorf("无法获取页面对象数量: %v", err)
		}

		for j := 0; j < objectCountRes.Count; j++ {
			objRes, err := instance.FPDFPage_GetObject(&requests.FPDFPage_GetObject{
				Page:  requests.Page{ByReference: &pdfPage.Page},
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
					Page:        requests.Page{ByReference: &pdfPage.Page},
				})
				if err != nil {
					return fmt.Errorf("无法获取图片元数据: %v", err)
				}
				fmt.Printf("图片元数据：%+v\n", imageMetadataRes.ImageMetadata)

				// imageDataDecodedRes, err := instance.FPDFImageObj_GetImageDataDecoded(&requests.FPDFImageObj_GetImageDataDecoded{
				// 	ImageObject: objRes.PageObject,
				// })
				// if err != nil {
				// 	return fmt.Errorf("无法获取图片解码数据: %v", err)
				// }

				// filePrefix := fmt.Sprintf("./images-files/decoded_%d_%d", i, j)
				// filename, err := util.SaveImageFromData(imageDataDecodedRes.Data, filePrefix)
				// if err != nil {
				// 	if strings.Contains(err.Error(), "无法解码图片") {
				// 		fmt.Printf("无法解码图片: %v\n", err)
				// 		continue
				// 	}
				// 	return fmt.Errorf("无法保存图片: %v", err)
				// }

				// defer os.Remove(filename)

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

				// _, err = instance.FPDFImageObj_LoadJpegFile(&requests.FPDFImageObj_LoadJpegFile{
				// 	ImageObject: objRes.PageObject,
				// 	Page: &requests.Page{
				// 		// ByReference: &pdfPage.Page,
				// 		ByIndex: &requests.PageByIndex{
				// 			Document: pdfDoc.Document,
				// 			Index:    i,
				// 		},
				// 	},
				// 	Count: 0,
				// 	// FileData: data,
				// 	FilePath: filename,
				// })
				// if err != nil {
				// 	fmt.Printf("FPDFImageObj_LoadJpegFileInline: %v\n", err)
				// 	return fmt.Errorf("无法加载图片: %v", err)
				// }

				// _, err = instance.FPDFPage_GenerateContent(&requests.FPDFPage_GenerateContent{
				// 	Page: requests.Page{
				// 		ByIndex: &requests.PageByIndex{
				// 			Document: pdfDoc.Document,
				// 			Index:    i,
				// 		},
				// 	},
				// })
				// if err != nil {
				// 	return err
				// }
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
		FilePath: &outputPath,
	})
	if err != nil {
		return fmt.Errorf("无法保存 PDF: %v", err)
	}

	util.CompareFileSize(inputPath, outputPath)

	return nil
}
