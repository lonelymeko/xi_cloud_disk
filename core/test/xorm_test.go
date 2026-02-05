package test

import (
	"bytes"
	"cloud_disk/core/models"
	"cloud_disk/core/utils"
	"encoding/json"
	"testing"

	_ "modernc.org/sqlite"

	"xorm.io/xorm"
)

// TestXorm 验证 Xorm 基础操作与表结构同步。
func TestXorm(t *testing.T) {
	engine, err := xorm.NewEngine("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	err = utils.EnsureSchema(engine)
	if err != nil {
		t.Fatal(err)
	}

	_, err = engine.InsertOne(&models.UserBasic{Name: "n", Email: "e", Password: "p", Identity: "id"})
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
	if dst.Len() == 0 {
		t.Fatal("empty json")
	}
}
