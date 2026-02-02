package global

import "xorm.io/xorm"

var (
	Engine = Init()
)

func Init() *xorm.Engine {
	engine, err := xorm.NewEngine("mysql", "root:12345678@tcp(127.0.0.1:3306)/cloud_disk?charset=utf8mb4&parseTime=True&loc=")
	if err != nil {
		panic(err)
		return nil
	}
	return engine
}
