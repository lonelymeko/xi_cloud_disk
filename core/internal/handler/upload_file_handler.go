// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package handler

import (
	"cloud_disk/core/common"
	"cloud_disk/core/internal/logic"
	"cloud_disk/core/internal/svc"
	"cloud_disk/core/internal/types"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"
	"crypto/md5"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func UploadFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UploadFileRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		file, fileHeader, err := r.FormFile("file")
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		defer file.Close()

		// 复制到系统临时文件夹
		tempFile, err := os.CreateTemp("", "upload-")
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		defer os.Remove(tempFile.Name())
		defer tempFile.Close()

		if _, err := io.Copy(tempFile, file); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 从临时文件计算 hash
		// 将文件指针重置到开头
		if _, err := tempFile.Seek(0, 0); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 初始化 MD5 哈希对象，分块读取文件内容更新哈希
		h := md5.New()
		// 定义分块大小（比如 4KB，可根据需求调整，避免内存溢出）
		buf := make([]byte, 4096)
		for {
			// 分块读取文件内容到 buf
			n, err := tempFile.Read(buf)
			if n > 0 {
				// 将读取到的有效字节更新到 MD5 哈希对象
				if _, err := h.Write(buf[:n]); err != nil {
					httpx.ErrorCtx(r.Context(), w, err)
					return
				}
			}
			if err == io.EOF {
				// 读取完毕，退出循环
				break
			}
			if err != nil {
				httpx.ErrorCtx(r.Context(), w, err)
				return
			}
		}

		// 生成 32 位小写十六进制 MD5 字符串
		md5Bytes := h.Sum(nil)
		hash := hex.EncodeToString(md5Bytes)

		// 判断文件是否已存在
		rp := new(models.RepositoryPool)
		has, err := svcCtx.DBEngine.Where("hash=?", hash).Get(rp)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		if has {
			// 文件已存在，秒传成功，返回文件信息
			httpx.OkJsonCtx(r.Context(), w, &types.UploadFileResponse{
				Identity: rp.Identity,
				Name:     rp.Name,
				Ext:      rp.Ext,
			})
			return
		}

		// 文件不存在，进行上传
		// 将临时文件指针重置到开头以便上传
		if _, err := tempFile.Seek(0, 0); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 判断是否为视频或图片文件，如果是则先压缩
		ext := strings.ToLower(path.Ext(fileHeader.Filename))
		videoExts := map[string]bool{
			".mp4": true, ".avi": true, ".mov": true, ".mkv": true,
			".flv": true, ".wmv": true, ".webm": true, ".m4v": true,
		}
		imageExts := map[string]bool{
			".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
			".bmp": true, ".webp": true,
		}

		var uploadFile *os.File
		var uploadFilename string
		var compressedFilePath string // 用于记录压缩文件路径，以便清理
		var actualSize int64          // 实际上传的文件大小
		var finalUploadPath string    // 最终要上传的文件路径（用于分片上传）

		if videoExts[ext] {
			// 是视频文件，需要压缩
			compressedFile, err := os.CreateTemp("", "compressed-*.mp4")
			if err != nil {
				httpx.ErrorCtx(r.Context(), w, err)
				return
			}
			compressedFilePath = compressedFile.Name()
			// 注意：不在这里 defer，避免在秒传时也执行清理

			// 使用 ffmpeg 压缩视频
			_, err = utils.CompressVideoWithFFmpeg(tempFile.Name(), compressedFile.Name(), 23, "128k")
			if err != nil {
				compressedFile.Close()
				os.Remove(compressedFilePath)
				httpx.ErrorCtx(r.Context(), w, err)
				return
			}

			// 使用压缩后的文件上传
			uploadFile = compressedFile
			uploadFilename = fileHeader.Filename
			finalUploadPath = compressedFile.Name()

			// 将文件指针重置到开头
			if _, err := uploadFile.Seek(0, 0); err != nil {
				compressedFile.Close()
				os.Remove(compressedFilePath)
				httpx.ErrorCtx(r.Context(), w, err)
				return
			}

			// 获取压缩后的文件大小
			fileInfo, err := uploadFile.Stat()
			if err != nil {
				compressedFile.Close()
				os.Remove(compressedFilePath)
				httpx.ErrorCtx(r.Context(), w, err)
				return
			}
			actualSize = fileInfo.Size()
		} else if imageExts[ext] {
			// 是图片文件，需要压缩
			compressedFile, err := os.CreateTemp("", "compressed-*"+ext)
			if err != nil {
				httpx.ErrorCtx(r.Context(), w, err)
				return
			}
			compressedFilePath = compressedFile.Name()
			tempCompressedPath := compressedFilePath
			compressedFile.Close() // 先关闭，因为 CompressImage 会重新打开

			// 使用图片压缩（最大 1920x1080，质量 85）
			err = utils.CompressImage(tempFile.Name(), tempCompressedPath, &utils.ImageCompressOptions{
				MaxWidth:  1920,
				MaxHeight: 1080,
				Quality:   85,
			})
			if err != nil {
				os.Remove(tempCompressedPath)
				httpx.ErrorCtx(r.Context(), w, err)
				return
			}

			// 重新打开压缩后的文件用于上传
			compressedFile, err = os.Open(tempCompressedPath)
			if err != nil {
				os.Remove(tempCompressedPath)
				httpx.ErrorCtx(r.Context(), w, err)
				return
			}

			// 使用压缩后的文件上传
			uploadFile = compressedFile
			uploadFilename = fileHeader.Filename
			finalUploadPath = tempCompressedPath

			// 获取压缩后的文件大小
			fileInfo, err := uploadFile.Stat()
			if err != nil {
				uploadFile.Close()
				os.Remove(tempCompressedPath)
				httpx.ErrorCtx(r.Context(), w, err)
				return
			}
			actualSize = fileInfo.Size()
		} else {
			// 非视频和图片文件，直接使用临时文件
			uploadFile = tempFile
			uploadFilename = fileHeader.Filename
			finalUploadPath = tempFile.Name()
			actualSize = fileHeader.Size // 使用原始文件大小
		}

		// 根据文件大小选择上传方式
		var OssPath string
		if actualSize > common.MultipartUploadThreshold {
			// 大文件：使用分片上传
			logx.Infof("文件大小 %.2f MB 超过阈值，使用分片上传",
				float64(actualSize)/(1024*1024))

			// 关闭文件句柄（分片上传会重新打开）
			if uploadFile != tempFile {
				uploadFile.Close()
			}

			OssPath, err = utils.UploadToOSSMultipart(finalUploadPath, uploadFilename, actualSize)
		} else {
			// 小文件：使用普通上传
			OssPath, err = utils.UploadToOSS(uploadFile, uploadFilename)
		}

		// 上传完成后，立即清理压缩文件
		if compressedFilePath != "" {
			uploadFile.Close()
			os.Remove(compressedFilePath)
		}

		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		// 文件上传成功，保存文件信息
		req.Ext = path.Ext(fileHeader.Filename)
		req.Size = actualSize // 使用实际上传的文件大小（压缩后）
		req.Name = fileHeader.Filename
		req.Path = OssPath
		req.Hash = hash

		l := logic.NewUploadFileLogic(r.Context(), svcCtx)
		resp, err := l.UploadFile(&req)
		common.Response(r, w, resp, err)
	}
}
