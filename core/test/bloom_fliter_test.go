package test

import (
	"fmt"
	"testing"

	"github.com/bits-and-blooms/bloom/v3"
)

func TestFilter(t *testing.T) {
	// 1. 初始化布隆过滤器
	// 参数说明：
	// - m: 过滤器的位数组大小（越大误判率越低，占用内存越多）
	// - k: 哈希函数的个数（需根据m和预期元素数量合理设置）
	// 推荐：预期存储n个元素，误判率p，可通过 bloom.EstimateParameters(n, p) 自动计算m和k
	n := uint(10000) // 预期存储10000个元素
	p := 0.01        // 可接受的误判率（1%）
	m, k := bloom.EstimateParameters(n, p)
	filter := bloom.New(m, k)

	// 2. 向过滤器中添加元素
	elements := []string{"apple", "banana", "orange", "grape"}
	for _, elem := range elements {
		filter.Add([]byte(elem)) // 布隆过滤器接收[]byte类型，需转换
		fmt.Printf("已添加元素: %s\n", elem)
	}

	// 3. 检测元素是否存在
	checkElements := []string{"apple", "banana", "watermelon", "grape"}
	for _, elem := range checkElements {
		exists := filter.Test([]byte(elem))
		if exists {
			fmt.Printf("元素 '%s' 可能存在（布隆过滤器检测）\n", elem)
		} else {
			fmt.Printf("元素 '%s' 一定不存在\n", elem)
		}
	}

}
