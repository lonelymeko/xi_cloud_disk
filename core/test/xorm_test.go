package test

import (
	"bytes"
	"cloud_disk/core/models"
	"encoding/json"
	"fmt"
	"testing"

	_ "github.com/go-sql-driver/mysql"

	"xorm.io/xorm"
)

func TestXorm(t *testing.T) {
	engine, err := xorm.NewEngine("mysql", "root:12345678@tcp(127.0.0.1:3306)/cloud_disk?charset=utf8mb4&parseTime=True&loc=")
	if err != nil {
		t.Fatal(err)
	}
	data := make([]models.UserBasic, 0)
	err = engine.Find(&data)
	if err != nil {
		t.Fatal(err)
	}
	// 将查询到的用户数据转换为JSON格式
	marshal, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	// 创建一个新的字节缓冲区用于格式化输出
	dst := new(bytes.Buffer)
	// 对JSON数据进行格式化，增加缩进以便于阅读
	err = json.Indent(dst, marshal, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	// 打印格式化后的JSON字符串到控制台
	fmt.Println(dst.String())
}
