package util

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log"
	"os"

	"github.com/klippa-app/go-pdfium/enums"
)

func CompareFileSize(filePath1 string, filePath2 string) {
	size1, _ := fileSize(filePath1)
	size2, _ := fileSize(filePath2)

	fmt.Printf("文件1大小：%dB, 文件2大小：%dB\n", size1, size2)
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

func SaveImageFromData(data []byte, filePath string) (string, error) {
	writeRawFile(filePath, data)
	// 使用 image.Decode 直接解码图片
	img, format, err := image.Decode(bytes.NewBuffer(data))
	if err != nil {
		return "", fmt.Errorf("无法解码图片: %v", err)

	}

	// 根据格式保存文件
	switch format {
	case "jpeg", "jpg":
		filePath = fmt.Sprintf("%s.jpeg", filePath)
		return filePath, writeJPEGFile(filePath, img)
	case "png":
		filePath = fmt.Sprintf("%s.png", filePath)
		return filePath, writePNGFile(filePath, img)
	case "gif":
		filePath = fmt.Sprintf("%s.gif", filePath)
		return filePath, writeGIFFile(filePath, img)
	default:
		filePath = fmt.Sprintf("%s.raw", filePath)
		return filePath, writeRawFile(filePath, data)
	}
}

func RenderImage(data []byte, width int, height int, stride int, format int) (image.Image, error) {

	switch enums.FPDF_BITMAP_FORMAT(format) {

	case enums.FPDF_BITMAP_FORMAT_GRAY:
		fmt.Println("GRAY GRAY GRAY GRAY GRAY")
		img := image.NewGray(image.Rect(0, 0, width, height))
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				img.Set(x, y, color.Gray{data[y*stride+x]})
			}
		}
		return img, nil

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

		return img, nil
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
				img.Set(x, y, color.RGBA{r, g, b, a})
			}
		}
		// arr := ExtractAlphaChannel(img)
		// PrintAlphaArray(arr)
		return img, nil

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
		return img, nil

	}

	return nil, fmt.Errorf("不支持的图片格式: %d", format)
}

func ConvertToPNG(width, height, stride int, data []byte, outputPath string, quality int, format int) error {
	// 创建一个 RGBA 图像
	img, err := RenderImage(data, width, height, stride, format)
	if err != nil {
		return err
	}

	// 创建输出文件
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 将图像编码为 png 格式并写入输出文件
	if err := png.Encode(outFile, img); err != nil {
		return err
	}

	log.Printf("png 图像已保存到: %s", outputPath)
	return nil
}

func ConvertToJPEG(width, height, stride int, data []byte, outputPath string, quality int, format int) error {
	// 创建一个 RGBA 图像
	img, err := RenderImage(data, width, height, stride, format)
	if err != nil {
		return err
	}

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

	log.Printf("JPEG 图像已保存到: %s", outputPath)
	return nil
}

func ConvertToGray(img image.Image) image.Image {
	// 创建一个新的灰度图像
	grayImg := image.NewGray(img.Bounds())

	// 遍历 RGBA 图像的每个像素
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			// 获取 RGBA 像素值
			r, g, b, _ := img.At(x, y).RGBA()

			// 将 RGBA 转换为灰度值
			// 使用加权平均法计算灰度值
			grayValue := uint8((r*299 + g*587 + b*114) / 1000 >> 8) // 右移8位以转换为 uint8

			// 设置灰度图像中的像素
			grayImg.Set(x, y, color.Gray{Y: grayValue})
		}
	}
	return grayImg
}

func writeJPEGFile(filename string, img image.Image) error {

	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	opts := &jpeg.Options{Quality: 60} // Adjust the quality as needed
	return jpeg.Encode(outFile, img, opts)
}

// 新增函数：处理 GIF 格式
func writeGIFFile(filename string, img image.Image) error {

	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	opts := &gif.Options{
		NumColors: 256,
	}
	return gif.Encode(outFile, img, opts) // 使用默认的 GIF 编码选项
}

func writePNGFile(filename string, img image.Image) error {

	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return png.Encode(outFile, img)
}

func writeRawFile(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0644)
}

func MergePNGFiles(imagePath string, maskPath string, outputPath string) error {
	// 打开图像文件
	imgFile, err := os.Open(imagePath)
	if err != nil {
		return err
	}
	defer imgFile.Close()

	// 解码图像
	img, _, err := image.Decode(imgFile)
	if err != nil {
		return err
	}

	// 打开遮罩文件
	maskFile, err := os.Open(maskPath)
	if err != nil {
		return err
	}
	defer maskFile.Close()

	// 解码遮罩
	mask, _, err := image.Decode(maskFile)
	if err != nil {
		return err
	}

	// 创建一个新的 RGBA 图像
	bounds := img.Bounds()
	mergedImg := image.NewRGBA(bounds)

	// 遍历每个像素，合并图像和遮罩
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			imgColor := img.At(x, y)
			maskColor := mask.At(x, y)

			// 获取遮罩的 alpha 值
			_, _, _, a := maskColor.RGBA()

			// 将图像的颜���与遮罩的 alpha 值结合
			r, g, b, _ := imgColor.RGBA()
			mergedImg.Set(x, y, color.RGBA{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)})
		}
	}

	// // 获取两个图像的边界
	// bounds1 := img.Bounds()
	// bounds2 := mask.Bounds()

	// // 创建一个新的图像，大小为两个图像的合并边界
	// mergedBounds := bounds1.Union(bounds2)
	// mergedImg := image.NewRGBA(mergedBounds)

	// // 将第二个图像绘制到合并图像上
	// draw.Draw(mergedImg, bounds2, mask, image.Point{}, draw.Over)

	// // 将第一个图像绘制到合并图像上
	// draw.Draw(mergedImg, bounds1, img, image.Point{}, draw.Over)

	// 创建输出文件
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 将合并后的图像编码为 PNG 格式并写入输出文件
	if err := png.Encode(outFile, mergedImg); err != nil {
		return err
	}

	log.Printf("合并后的图像已保存到: %s", outputPath)
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
