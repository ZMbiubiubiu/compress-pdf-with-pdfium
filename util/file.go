package util

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
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
	// 使用 image.Decode 直接解码图片
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		// 临时写到文件中，留待后续处理
		writeRawFile(filePath, data)
		return "", fmt.Errorf("无法解码图片: %v", err)
	}

	// 根据格式保存文件
	switch format {
	case "jpeg":
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

func writeJPEGFile(filename string, img image.Image) error {

	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	opts := &jpeg.Options{Quality: 80} // Adjust the quality as needed
	return jpeg.Encode(outFile, img, opts)
}

// 新增函数：处理 GIF 格式
func writeGIFFile(filename string, img image.Image) error {

	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return gif.Encode(outFile, img, nil)
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
