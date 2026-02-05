package global

import (
	"cloud_disk/core/common"
	"fmt"
	"time"

	"github.com/rabbitmq/amqp091-go"
	"github.com/zeromicro/go-zero/core/logx"
)

// 全局 RabbitMQ 连接（包级别复用）
var RmqConn *amqp091.Connection

// 全局通道（简单场景可复用单通道，高并发建议通道池）
var RmqCh *amqp091.Channel

func InitRabbitMQ(host string, port int, username, password, vhost string) (*amqp091.Connection, *amqp091.Channel) {
	var err error
	RmqConn, err = amqp091.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d%s", username, password, host, port, vhost))
	if err != nil {
		panic(fmt.Sprintf("Failed to connect to RabbitMQ: %v", err))
	}
	// 2. 监听连接关闭（可选，用于重连）
	go func() {
		<-RmqConn.NotifyClose(make(chan *amqp091.Error))
		fmt.Println("RabbitMQ 连接断开，触发重连逻辑")
		// 重连逻辑（生产环境建议加）
		for {
			time.Sleep(3 * time.Second)
			if err := reinitRabbitMQ(host, port, username, password, vhost); err == nil {
				break
			}
		}
	}()
	RmqCh, err = RmqConn.Channel()
	if err != nil {
		panic(fmt.Sprintf("Failed to create RabbitMQ channel: %v", err))
	}

	// 3. 声明交换机和队列
	if err := declareRabbitMQResources(); err != nil {
		panic(fmt.Sprintf("Failed to declare RabbitMQ resources: %v", err))
	}

	return RmqConn, RmqCh

}

// 重连逻辑（可选，生产环境必备）
func reinitRabbitMQ(host string, port int, username, password, vhost string) error {
	newConn, err := amqp091.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d%s", username, password, host, port, vhost))
	if err != nil {
		return err
	}
	newCh, err := newConn.Channel()
	if err != nil {
		_ = newConn.Close()
		return err
	}

	// 替换全局连接/通道
	RmqConn = newConn
	RmqCh = newCh

	// 重新声明交换机/队列（幂等，不影响已存在的资源）
	_ = declareRabbitMQResources()
	return nil
}

// 声明交换机/队列（幂等，可重复调用）
func declareRabbitMQResources() error {
	// 1. 声明交换机（direct 类型，持久化）
	err := RmqCh.ExchangeDeclare(
		common.ExchangeName, // 交换机名
		"direct",            // 类型
		true,                // 持久化
		false,               // 非自动删除
		false,               // 非内部交换机
		false,               // 无等待
		nil,                 // 额外参数
	)
	if err != nil {
		return fmt.Errorf("声明交换机失败: %w", err)
	}

	// 2. 声明队列（按业务阶段拆分，均为持久化）
	queues := []string{
		common.QueueName,
	}
	for _, queue := range queues {
		_, err := RmqCh.QueueDeclare(
			queue, // 队列名
			true,  // 持久化
			false, // 非自动删除
			false, // 非排他
			false, // 无等待
			nil,   // 额外参数（可配置死信队列）
		)
		if err != nil {
			return fmt.Errorf("声明队列 %s 失败: %w", queue, err)
		}

		// 3. 绑定队列到交换机（路由键与队列名一致）
		err = RmqCh.QueueBind(
			queue,               // 队列名
			common.RoutingKey,   // 路由键
			common.ExchangeName, // 交换机名
			false,               // 无等待
			nil,                 // 额外参数
		)
		if err != nil {
			return fmt.Errorf("绑定队列 %s 失败: %w", queue, err)
		}
	}
	logx.Infof("RabbitMQ 资源声明成功: 交换机 %s, 队列 %v", common.ExchangeName, queues)

	return nil
}
