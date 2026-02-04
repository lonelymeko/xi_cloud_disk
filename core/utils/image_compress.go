package utils

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

// ImageCompressOptions 图片压缩选项
type ImageCompressOptions struct {
	MaxWidth  int // 最大宽度（像素），0 表示不限制
	MaxHeight int // 最大高度（像素），0 表示不限制
	Quality   int // JPEG 质量 (1-100)，PNG 忽略此参数
}

// DefaultImageCompressOptions 默认图片压缩选项
var DefaultImageCompressOptions = ImageCompressOptions{
	MaxWidth:  1920, // 最大宽度 1920px
	MaxHeight: 1080, // 最大高度 1080px
	Quality:   85,   // JPEG 质量 85
}

// CompressImage 压缩图片
// inputPath: 输入图片路径
// outputPath: 输出图片路径
// options: 压缩选项（可选，传 nil 使用默认值）
func CompressImage(inputPath, outputPath string, options *ImageCompressOptions) error {
	// 1. 参数校验
	if inputPath == "" || outputPath == "" {
		return errors.New("输入/输出文件路径不能为空")
	}

	// 2. 使用默认选项
	if options == nil {
		options = &DefaultImageCompressOptions
	}

	// 3. 打开输入文件
	inputFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("打开输入文件失败: %w", err)
	}
	defer inputFile.Close()

	// 4. 解码图片
	img, _, err := image.Decode(inputFile)
	if err != nil {
		return fmt.Errorf("解码图片失败: %w", err)
	}

	// 5. 获取原始尺寸
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 6. 计算缩放后的尺寸
	newWidth, newHeight := calculateResizeSize(width, height, options.MaxWidth, options.MaxHeight)

	// 7. 如果需要缩放
	var resizedImg image.Image
	if newWidth != width || newHeight != height {
		resizedImg = resizeImage(img, newWidth, newHeight)
	} else {
		resizedImg = img
	}

	// 8. 创建输出文件
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outputFile.Close()

	// 9. 根据格式编码输出
	ext := strings.ToLower(filepath.Ext(outputPath))
	switch ext {
	case ".jpg", ".jpeg":
		// JPEG 格式，支持质量设置
		err = jpeg.Encode(outputFile, resizedImg, &jpeg.Options{Quality: options.Quality})
	case ".png":
		// PNG 格式
		err = png.Encode(outputFile, resizedImg)
	default:
		// 默认使用 JPEG
		err = jpeg.Encode(outputFile, resizedImg, &jpeg.Options{Quality: options.Quality})
	}

	if err != nil {
		return fmt.Errorf("编码图片失败: %w", err)
	}

	return nil
}

// CompressImageFromReader 从 io.Reader 压缩图片到 io.Writer
// 适用于不需要保存到文件的场景（如直接上传到 OSS）
func CompressImageFromReader(reader io.Reader, writer io.Writer, format string, options *ImageCompressOptions) error {
	// 1. 使用默认选项
	if options == nil {
		options = &DefaultImageCompressOptions
	}

	// 2. 解码图片
	img, _, err := image.Decode(reader)
	if err != nil {
		return fmt.Errorf("解码图片失败: %w", err)
	}

	// 3. 获取原始尺寸
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// 4. 计算缩放后的尺寸
	newWidth, newHeight := calculateResizeSize(width, height, options.MaxWidth, options.MaxHeight)

	// 5. 如果需要缩放
	var resizedImg image.Image
	if newWidth != width || newHeight != height {
		resizedImg = resizeImage(img, newWidth, newHeight)
	} else {
		resizedImg = img
	}

	// 6. 根据格式编码输出
	format = strings.ToLower(format)
	switch format {
	case ".jpg", ".jpeg", "jpg", "jpeg":
		err = jpeg.Encode(writer, resizedImg, &jpeg.Options{Quality: options.Quality})
	case ".png", "png":
		err = png.Encode(writer, resizedImg)
	default:
		// 默认使用 JPEG
		err = jpeg.Encode(writer, resizedImg, &jpeg.Options{Quality: options.Quality})
	}

	if err != nil {
		return fmt.Errorf("编码图片失败: %w", err)
	}

	return nil
}

// calculateResizeSize 计算缩放后的尺寸（保持宽高比）
func calculateResizeSize(width, height, maxWidth, maxHeight int) (int, int) {
	// 如果没有设置最大尺寸限制，返回原始尺寸
	if maxWidth <= 0 && maxHeight <= 0 {
		return width, height
	}

	// 如果原始尺寸小于最大尺寸，不需要缩放
	if (maxWidth <= 0 || width <= maxWidth) && (maxHeight <= 0 || height <= maxHeight) {
		return width, height
	}

	// 计算缩放比例
	var scale float64 = 1.0

	if maxWidth > 0 && width > maxWidth {
		scale = float64(maxWidth) / float64(width)
	}

	if maxHeight > 0 && height > maxHeight {
		heightScale := float64(maxHeight) / float64(height)
		if heightScale < scale {
			scale = heightScale
		}
	}

	// 计算新尺寸
	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	return newWidth, newHeight
}

// resizeImage 缩放图片
func resizeImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	// 使用高质量的 Lanczos3 算法进行缩放
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}

// GetImageInfo 获取图片信息
func GetImageInfo(filePath string) (width, height int, format string, err error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, 0, "", err
	}
	defer file.Close()

	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, "", err
	}

	return config.Width, config.Height, format, nil
}

// CompressImageToBytes 将图片压缩为字节数组
func CompressImageToBytes(inputPath string, options *ImageCompressOptions) ([]byte, error) {
	var buf bytes.Buffer

	inputFile, err := os.Open(inputPath)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()

	format := filepath.Ext(inputPath)
	err = CompressImageFromReader(inputFile, &buf, format, options)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
