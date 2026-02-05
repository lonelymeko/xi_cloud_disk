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

		// 从 fileHeader 获取文件名和大小（如果 req 中没有提供）
		if req.Name == "" {
			req.Name = fileHeader.Filename
		}
		if req.Size == 0 {
			req.Size = fileHeader.Size
		}
		// 从文件名提取扩展名（如果 req 中没有提供）
		if req.Ext == "" {
			// 获取文件扩展名
			for i := len(fileHeader.Filename) - 1; i >= 0; i-- {
				if fileHeader.Filename[i] == '.' {
					req.Ext = fileHeader.Filename[i:]
					break
				}
			}
		}

		// 调试日志
		logx.Infof("文件上传信息 - Name: %s, Ext: %s, Size: %d", req.Name, req.Ext, req.Size)

		// 复制到临时文件夹
		tempFile, err := os.OpenFile("/tmp/upload-"+utils.UUID()+req.Ext, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o644)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		defer tempFile.Close()

		if _, copyErr := io.Copy(tempFile, file); copyErr != nil {
			httpx.ErrorCtx(r.Context(), w, copyErr)
			return
		}

		// 从临时文件计算 hash
		// 将文件指针重置到开头
		if _, seekErr := tempFile.Seek(0, 0); seekErr != nil {
			httpx.ErrorCtx(r.Context(), w, seekErr)
			return
		}

		// 初始化 MD5 哈希对象，分块读取文件内容更新哈希
		h := md5.New()
		// 定义分块大小（比如 4KB，可根据需求调整，避免内存溢出）
		buf := make([]byte, 4096)
		for {
			// 分块读取文件内容到 buf
			n, readErr := tempFile.Read(buf)
			if n > 0 {
				// 将读取到的有效字节更新到 MD5 哈希对象
				if _, writeErr := h.Write(buf[:n]); writeErr != nil {
					httpx.ErrorCtx(r.Context(), w, writeErr)
					return
				}
			}
			if readErr == io.EOF {
				// 读取完毕，退出循环
				break
			}
			if readErr != nil {
				httpx.ErrorCtx(r.Context(), w, readErr)
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

		// 如果文件信息存在：
		if has {
			l := logic.NewUploadFileLogic(r.Context(), svcCtx)
			resp, err := l.UploadFile(&req, has, rp.Identity, tempFile.Name(), hash)
			common.Response(r, w, resp, err)
			return
		}
		// 不存在
		identity := utils.UUID()
		l := logic.NewUploadFileLogic(r.Context(), svcCtx)
		resp, err := l.UploadFile(&req, has, identity, tempFile.Name(), hash)
		common.Response(r, w, resp, err)
	}
}
