package util

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"io"
	"os"
	"path/filepath"

	"github.com/klippa-app/go-pdfium/enums"
	"github.com/nfnt/resize"
)

func CompareFileSize(filePath1 string, filePath2 string) {
	size1, _ := fileSize(filePath1)
	size2, _ := fileSize(filePath2)

	fmt.Printf("文件1大小：%.fKB, 文件2大小：%.fKB\n", float64(size1)/1024, float64(size2)/1024)
	fmt.Printf("文件2/文件1：%.2f%%\n", (float64(size2)/float64(size1))*100)
}

func fileSize(filePath string) (int64, error) {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}
	return fileInfo.Size(), nil
}

func CopyFile(srcPath string, dstPath string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func RenderImage(data []byte, width int, height int, stride int, format int) (isAlphaValid bool, img image.Image, err error) {

	switch enums.FPDF_BITMAP_FORMAT(format) {

	case enums.FPDF_BITMAP_FORMAT_GRAY:
		fmt.Println("GRAY GRAY GRAY GRAY GRAY")
		img := image.NewGray(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				img.Set(x, y, color.Gray{data[y*stride+x]})
			}
		}
		return false, img, nil

	case enums.FPDF_BITMAP_FORMAT_BGR:
		fmt.Println("BGR BGR BGR BGR BGR")
		img := image.NewRGBA(image.Rect(0, 0, width, height))

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {

				var r, g, b, a uint8
				// 计算数据索引
				index := y*stride + x*3 // 每个像素有 3 个字节（BGR）

				if index+2 < len(data) { // 确保不越界
					b = data[index]
					g = data[index+1]
					r = data[index+2]
					a = 255 // 默认 alpha 为 255
				}
				img.Set(x, y, color.RGBA{r, g, b, a})
			}
		}
		return false, img, nil

	case enums.FPDF_BITMAP_FORMAT_BGRA:
		fmt.Println("BGRA BGRA BGRA BGRA BGRA")
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {

				var r, g, b, a uint8
				// 计算数据索引
				index := y*stride + x*4 // 假设每个像素有 4 个字节（RGBA）

				if index+3 < len(data) { // 确保不越界
					b = data[index]
					g = data[index+1]
					r = data[index+2]
					a = data[index+3]
				}
				if a != 255 {
					isAlphaValid = true
				}
				img.Set(x, y, color.RGBA{r, g, b, a})
			}
		}
		// arr := ExtractAlphaChannel(img)
		// PrintAlphaArray(arr)
		return isAlphaValid, img, nil

	case enums.FPDF_BITMAP_FORMAT_BGRX:
		fmt.Println("BGRX BGRX BGRX BGRX BGRX")
		img := image.NewRGBA(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				var r, g, b, a uint8

				// 计算数据索引
				index := y*stride + x*4 // 假设每个像素有 4 个字节（RGBA）

				if index+3 < len(data) { // 确保不越界
					b = data[index]
					g = data[index+1]
					r = data[index+2]
					a = 255
				}
				img.Set(x, y, color.RGBA{r, g, b, a})
			}
		}
		return false, img, nil
	}

	return false, nil, fmt.Errorf("不支持的图片格式: %d", format)
}

func ConvertToJPEG(img image.Image, outputPath string, quality int) error {

	// 创建输出文件
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 设置 JPEG 压缩质量
	jpegOptions := &jpeg.Options{Quality: quality}

	// 将图像编码为 JPEG 格式并写入输出文件
	if err := jpeg.Encode(outFile, img, jpegOptions); err != nil {
		return err
	}

	return nil
}

// ExtractAlphaChannel 从 RGBA 图像中提取 alpha 通道并存储在二维数组中
func ExtractAlphaChannel(img *image.RGBA) [][]uint8 {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	// 创建一个二维数组来存储 alpha 值
	alphaArray := make([][]uint8, height)
	for i := range alphaArray {
		alphaArray[i] = make([]uint8, width)
	}

	// 填充 alpha 值
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// 获取当前像素的颜色
			c := img.RGBAAt(x, y)
			alphaArray[y][x] = c.A // 存储 alpha 值
		}
	}

	return alphaArray
}

// 打印二维数组
func PrintAlphaArray(alphaArray [][]uint8) {
	for _, row := range alphaArray {
		for _, a := range row {
			fmt.Printf("%d ", a) // 打印 alpha 值
		}
		fmt.Println() // 换行
	}
}

// ChangeDPI 修改图像的 DPI
func ReduceDPI(img image.Image, originalWidth int, originalDPI, newDPI float32) image.Image {

	// 获取原始图像的尺寸
	width := uint(float32(originalWidth) * newDPI / originalDPI)
	if width > uint(img.Bounds().Dx()) {
		return img
	}
	fmt.Printf("original width: %d, new width: %d originalDPI: %f, newDPI: %f\n", img.Bounds().Dx(), width, originalDPI, newDPI)

	return resize.Resize(width, 0, img, resize.Bicubic)
}

func GetFilePath(inputDir string, fileExt string) []string {
	var pdfPaths []string

	// 使用 Walk 函数遍历目录
	filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 检查文件扩展名是否为 .pdf
		if !info.IsDir() && (filepath.Ext(path) != "" && filepath.Ext(path) == fileExt) {
			pdfPaths = append(pdfPaths, path)
		}
		return nil
	})

	return pdfPaths
}
