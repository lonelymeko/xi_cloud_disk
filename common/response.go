package common

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
)

type Body struct {
	Code uint32      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// Response 统一响应处理
func Response(r *http.Request, w http.ResponseWriter, resp interface{}, err error) {
	if err == nil {
		// 成功响应
		r := Body{
			Code: 0,
			Msg:  "ok",
			Data: resp,
		}
		httpx.WriteJson(w, http.StatusOK, r)
		return
	}

	// 错误响应
	errCode := uint32(404)
	// 可根据错误类型，返回具体错误信息
	errMsg := err.Error()
	httpx.WriteJson(w, http.StatusBadRequest, Body{
		Code: errCode,
		Msg:  errMsg,
		Data: nil,
	})
}
