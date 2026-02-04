package utils

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
)

// CompressVideoWithFFmpeg 调用 ffmpeg 压缩视频（基于 H.264 编码，兼顾画质和体积）
// inputPath: 输入视频文件路径（如 "./source/input.mp4"）
// outputPath: 输出压缩视频文件路径（如 "./output/compressed.mp4"）
// crf: 画质控制参数（0-51，推荐 20-28，23 为默认最优）
// audioBitrate: 音频码率（如 "128k"、"96k"）
// 返回值: 命令执行输出信息（成功时为空）、错误信息
func CompressVideoWithFFmpeg(inputPath, outputPath string, crf int, audioBitrate string) (string, error) {
	// 1. 校验必要参数的合法性
	if inputPath == "" || outputPath == "" {
		return "", errors.New("输入/输出文件路径不能为空")
	}
	if crf < 0 || crf > 51 {
		return "", errors.New("CRF 值必须在 0-51 之间")
	}
	if audioBitrate == "" {
		audioBitrate = "128k" // 默认音频码率
	}

	// 2. 构造 ffmpeg 命令参数
	// 对应指令: ffmpeg -i input.mp4 -c:v libx264 -crf 23 -c:a aac -b:a 128k output.mp4
	cmdArgs := []string{
		"-i", inputPath, // 指定输入文件
		"-c:v", "libx264", // 视频编码器使用 H.264
		"-crf", fmt.Sprintf("%d", crf), // 画质控制参数
		"-c:a", "aac", // 音频编码器使用 AAC
		"-b:a", audioBitrate, // 音频码率
		"-y",       // 覆盖已存在的输出文件（无需手动确认，批量处理时实用）
		outputPath, // 指定输出文件
	}

	// 3. 构建执行命令
	cmd := exec.Command("ffmpeg", cmdArgs...)

	// 4. 捕获命令执行的标准输出和错误输出（方便排查问题）
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// 5. 执行命令并等待完成
	err := cmd.Run()

	// 6. 整理执行结果
	output := fmt.Sprintf("标准输出: %s\n错误输出: %s", stdoutBuf.String(), stderrBuf.String())
	if err != nil {
		return output, fmt.Errorf("ffmpeg 命令执行失败: %w, 执行详情: %s", err, output)
	}

	// 7. 执行成功，返回空输出和 nil 错误
	return "", nil
}
