package global

import (
	_ "github.com/go-sql-driver/mysql"
	"xorm.io/xorm"
)

func Init(dataSource string) *xorm.Engine {
	engine, err := xorm.NewEngine("mysql", dataSource)
	if err != nil {
		panic(err)
	}
	return engine
}
