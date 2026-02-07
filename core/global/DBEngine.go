package global

import (
	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
)

// Init 初始化数据库引擎。
func Init(dataSource string) *xorm.Engine {
	engine, err := xorm.NewEngine("mysql", dataSource)
	if err != nil {
		panic(err)
	}
	return engine
}
