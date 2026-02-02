// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package main

import (
	"cloud_disk/common"
	"cloud_disk/core/internal/config"
	"cloud_disk/core/internal/handler"
	"cloud_disk/core/internal/svc"
	"flag"
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "core/etc/core-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf, rest.WithUnauthorizedCallback(JwtUnauthorizedResult))
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

// JwtUnauthorizedResult 鉴权失败
func JwtUnauthorizedResult(w http.ResponseWriter, r *http.Request, err error) {
	fmt.Println("JwtUnauthorizedResult:", err)
	httpx.WriteJson(w, http.StatusOK, common.Body{
		Code: 401,
		Msg:  "鉴权失败",
		Data: nil,
	})
}
