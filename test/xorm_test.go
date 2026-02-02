package test

import (
	"testing"

	"xorm.io/xorm"
)

func TestXorm(t *testing.T) {
	engine, err := xorm.NewEngine("mysql", "r oot:root@tcp(127.0.0.1:3306)/zero_db?charset=utf8mb4&parseTime=True&loc=")
	if err != nil {
		t.Fatal(err)
	}
	engine.Find()
}
