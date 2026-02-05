package common

var OSSRegion = "cn-chengdu"
var OSSBucketName = "xi-cloud-disk-chengdu"

// 分页的默认参数
var PageSize = 20

var DataTimeFormat = "2006-01-02 15:04:05"

// 分片上传配置
const (
	// PartSize 分片大小：10MB（阿里云 OSS 推荐 5MB-10MB）
	PartSize = 10 * 1024 * 1024

	// MultipartUploadThreshold 超过此大小使用分片上传：100MB
	MultipartUploadThreshold = 100 * 1024 * 1024

	// MaxConcurrentParts 最大并发上传分片数
	MaxConcurrentParts = 3
)

// RabbitMq 配置
var ExchangeName = "upload.event.exchange"

var QueueName = "upload.process.queue"

var RoutingKey = "upload.new"
