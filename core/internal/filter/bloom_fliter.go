package filter

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/bits-and-blooms/bloom/v3"
	"xorm.io/xorm"
)

type MyBloomFilter struct {
	filter *bloom.BloomFilter
	mutex  sync.RWMutex // 读写锁：读多写少场景更高效
}

const (
	filterFile = "./bloom_filter.data" // 持久化文件路径
	n          = 10000                 // 预期元素数
	p          = 0.01                  // 误判率
)

func NewBloomFilter(eng *xorm.Engine) *MyBloomFilter {
	m, k := bloom.EstimateParameters(n, p)
	bloomFilter := &MyBloomFilter{
		filter: bloom.New(m, k),
	}
	bloomFilter.initBloomFilter(eng)

	return bloomFilter
}

// 初始化布隆过滤器（优先从文件加载，加载失败则新建）
func (b *MyBloomFilter) initBloomFilter(eng *xorm.Engine) {
	// 1. 尝试从文件加载
	file, err := os.Open(filterFile)
	if err == nil {
		defer file.Close()
		_, err := b.filter.ReadFrom(file)
		if err == nil {
			log.Println("布隆过滤器从文件加载成功")
			return
		} else {
			log.Printf("从文件加载布隆过滤器失败: %v\n", err)
		}
	} else {
		log.Printf("无法打开布隆过滤器文件: %v\n", err)
	}

	// 2. 加载失败，新建过滤器
	m, k := bloom.EstimateParameters(n, p)
	b.filter = bloom.New(m, k)
	log.Println("创建新的布隆过滤器")

	// 如果是首次启动，需要批量插入初始key
	err = b.batchAddKeys(eng)
	if err != nil {
		panic(fmt.Sprintf("初始化布隆过滤器失败: %v", err))
	}

	// 3. 持久化到文件
	b.saveFilterToFile()
	log.Println("布隆过滤器初始化完成并已保存到文件")
}

// SaveFilterToFile 保存过滤器到文件（公共方法）
func (b *MyBloomFilter) SaveFilterToFile() {
	b.saveFilterToFile()
}

// 保存过滤器到文件（私有方法）
func (b *MyBloomFilter) saveFilterToFile() {
	file, err := os.Create(filterFile)
	if err != nil {
		log.Printf("创建布隆过滤器文件失败: %v\n", err)
		return
	}
	defer file.Close()
	_, err = b.filter.WriteTo(file)
	if err != nil {
		log.Printf("保存布隆过滤器到文件失败: %v\n", err)
		return
	}
	log.Println("布隆过滤器已保存到文件")
}

// 高并发写入加锁
func (b *MyBloomFilter) AddFileHash(hash string) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	b.filter.Add([]byte(hash))
}

// 检测Hash是否存在（读操作用读锁）
func (b *MyBloomFilter) IsFileExisted(uuid string) bool {
	b.mutex.RLock()
	defer b.mutex.RUnlock()
	return b.filter.Test([]byte(uuid))
}

func (b *MyBloomFilter) batchAddKeys(eng *xorm.Engine) error {
	// 查询所有hash值
	type HashResult struct {
		Hash string
	}
	var results []HashResult
	err := eng.Table("repository_pool").Select("hash").Find(&results)
	if err != nil {
		return fmt.Errorf("查询hash失败: %w", err)
	}

	// 提取hash值
	hashs := make([]string, 0, len(results))
	for _, result := range results {
		if result.Hash != "" {
			hashs = append(hashs, result.Hash)
		}
	}

	if len(hashs) == 0 {
		log.Println("数据库中暂无文件hash信息")
		return nil
	}
	b.mutex.Lock()
	// 批量添加到布隆过滤器
	for _, hash := range hashs {
		if hash != "" { // 过滤空值
			b.filter.Add([]byte(hash))
			fmt.Printf("已添加hash到布隆过滤器: %s\n", hash)
		}
	}
	b.mutex.Unlock()
	log.Printf("成功从数据库加载 %d 个hash到布隆过滤器\n", len(hashs))
	return nil
}
