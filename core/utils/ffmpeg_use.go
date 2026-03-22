package utils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

var ffmpegEncodeLimiter = make(chan struct{}, 1)

func getFFmpegThreads() int {
	cpu := runtime.NumCPU()
	if cpu <= 1 {
		return 1
	}
	threads := cpu / 2
	if threads < 1 {
		threads = 1
	}
	if threads > 4 {
		threads = 4
	}
	return threads
}

func getFFmpegTimeout(inputPath string) time.Duration {
	fi, err := os.Stat(inputPath)
	if err != nil {
		return 20 * time.Minute
	}
	// 50MB 约给 8 分钟，避免无限卡住；大文件按体积放宽。
	base := 8 * time.Minute
	extra := time.Duration(fi.Size()/(50*1024*1024)) * 6 * time.Minute
	t := base + extra
	if t < 8*time.Minute {
		return 8 * time.Minute
	}
	if t > 45*time.Minute {
		return 45 * time.Minute
	}
	return t
}

// CompressVideoWithFFmpeg 调用 ffmpeg 压缩视频（基于 H.264 编码，兼顾画质和体积）。
// inputPath: 输入视频文件路径（如 "./source/input.mp4"）。
// outputPath: 输出压缩视频文件路径（如 "./output/compressed.mp4"）。
// crf: 画质控制参数（0-51，推荐 20-28，23 为默认最优）。
// audioBitrate: 音频码率（如 "128k"、"96k"）。
// 返回值: 命令执行输出信息（成功时为空）、错误信息。
func CompressVideoWithFFmpeg(inputPath, outputPath string, crf int, audioBitrate string) (string, error) {
	// 1. 校验必要参数的合法性
	if inputPath == "" || outputPath == "" {
		return "", errors.New("输入/输出文件路径不能为空")
	}
	if crf < 0 || crf > 51 {
		return "", errors.New("CRF 值必须在 0-51 之间")
	}
	if crf < 24 {
		crf = 24
	}
	if crf > 30 {
		crf = 30
	}
	if audioBitrate == "" {
		audioBitrate = "96k" // 默认音频码率，优先速度与体积
	}

	ffmpegEncodeLimiter <- struct{}{}
	defer func() { <-ffmpegEncodeLimiter }()

	ctx, cancel := context.WithTimeout(context.Background(), getFFmpegTimeout(inputPath))
	defer cancel()

	threads := getFFmpegThreads()

	// 2. 构造 ffmpeg 命令参数
	// 对应指令: ffmpeg -i input.mp4 -c:v libx264 -crf 23 -c:a aac -b:a 128k output.mp4
	cmdArgs := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-nostdin",
		"-threads", strconv.Itoa(threads),
		"-i", inputPath,
		"-map_metadata", "-1",
		"-map_chapters", "-1",
		"-vf", "scale=1280:-2:force_original_aspect_ratio=decrease",
		"-c:v", "libx264",
		"-preset", "veryfast",
		"-crf", fmt.Sprintf("%d", crf),
		"-maxrate", "2200k",
		"-bufsize", "4400k",
		"-pix_fmt", "yuv420p",
		"-c:a", "aac",
		"-b:a", audioBitrate,
		"-movflags", "+faststart",
		"-y",
		outputPath,
	}

	// 3. 构建执行命令
	cmd := exec.CommandContext(ctx, "ffmpeg", cmdArgs...)

	// 4. 捕获命令执行的标准输出和错误输出（方便排查问题）
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	// 5. 执行命令并等待完成
	err := cmd.Run()

	// 6. 整理执行结果
	output := fmt.Sprintf("标准输出: %s\n错误输出: %s", stdoutBuf.String(), stderrBuf.String())
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return output, fmt.Errorf("ffmpeg 压缩超时，已中止: %s", output)
		}
		return output, fmt.Errorf("ffmpeg 命令执行失败: %w, 执行详情: %s", err, output)
	}

	// 7. 执行成功，返回空输出和 nil 错误
	return "", nil
}
