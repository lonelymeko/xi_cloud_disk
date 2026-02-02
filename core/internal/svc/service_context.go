// Code scaffolded by goctl. Safe to edit.
// goctl 1.9.2

package svc

import (
	"cloud_disk/core/internal/config"
	"cloud_disk/global"

	"xorm.io/xorm"
)

type ServiceContext struct {
	Config   config.Config
	DBEngine *xorm.Engine
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:   c,
		DBEngine: global.Engine,
	}
}
