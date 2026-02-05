package test

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	amqp "github.com/rabbitmq/amqp091-go"
)

func rabbitmqURL(t *testing.T) string {
	t.Helper()
	_ = godotenv.Load("core/.env")
	host := os.Getenv("RABBITMQ_HOST")
	port := os.Getenv("RABBITMQ_PORT")
	user := os.Getenv("RABBITMQ_USERNAME")
	pass := os.Getenv("RABBITMQ_PASSWORD")
	vhost := os.Getenv("RABBITMQ_VHOST")
	if host == "" || port == "" || user == "" {
		t.Skip("RabbitMQ env not set")
	}
	if vhost == "" {
		vhost = "/"
	}
	if !strings.HasPrefix(vhost, "/") {
		vhost = "/" + vhost
	}
	return fmt.Sprintf("amqp://%s:%s@%s:%s%s", user, pass, host, port, vhost)
}

func uniqueName(t *testing.T, prefix string) string {
	t.Helper()
	safe := strings.ReplaceAll(t.Name(), "/", "_")
	return fmt.Sprintf("%s_%s_%d", prefix, safe, time.Now().UnixNano())
}

// TestRabbitMQConnection 测试 RabbitMQ 连接
func TestRabbitMQConnection(t *testing.T) {
	conn, err := amqp.Dial(rabbitmqURL(t))
	if err != nil {
		t.Fatalf("无法连接到 RabbitMQ: %v", err)
	}
	defer conn.Close()

	t.Log("成功连接到 RabbitMQ")
}

// TestRabbitMQChannel 测试创建 Channel
func TestRabbitMQChannel(t *testing.T) {
	conn, err := amqp.Dial(rabbitmqURL(t))
	if err != nil {
		t.Fatalf("无法连接到 RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("无法创建 Channel: %v", err)
	}
	defer ch.Close()

	t.Log("成功创建 Channel")
}

// TestRabbitMQDeclareQueue 测试声明队列
func TestRabbitMQDeclareQueue(t *testing.T) {
	conn, err := amqp.Dial(rabbitmqURL(t))
	if err != nil {
		t.Fatalf("无法连接到 RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("无法创建 Channel: %v", err)
	}
	defer ch.Close()

	qName := uniqueName(t, "test_queue")
	q, err := ch.QueueDeclare(
		qName, // 队列名称
		false, // durable（持久化）：false 表示队列仅存于内存，RabbitMQ 重启后丢失
		false, // delete when unused（未使用时删除）：false 表示即使无消费者，队列也不删除
		false, // exclusive（排他性）：false 表示多个连接可访问该队列
		false, // no-wait（非阻塞）：false 表示等待服务器返回声明成功的确认
		nil,   // arguments（自定义参数）：nil 表示使用默认配置
	)
	if err != nil {
		t.Fatalf("无法声明队列: %v", err)
	}

	t.Logf("成功声明队列: %s", q.Name)
}

// TestRabbitMQPublishAndConsume 测试发布和消费消息
func TestRabbitMQPublishAndConsume(t *testing.T) {
	conn, err := amqp.Dial(rabbitmqURL(t))
	if err != nil {
		t.Fatalf("无法连接到 RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("无法创建 Channel: %v", err)
	}
	defer ch.Close()

	// 声明队列
	qName := uniqueName(t, "test_queue")
	q, err := ch.QueueDeclare(
		qName,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("无法声明队列: %v", err)
	}

	// 发布消息
	body := "Hello RabbitMQ Test"
	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
	if err != nil {
		t.Fatalf("无法发布消息: %v", err)
	}
	t.Logf("成功发布消息: %s", body)

	// 消费消息
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		t.Fatalf("无法注册消费者: %v", err)
	}

	// 接收消息
	msg := <-msgs
	t.Logf("成功消费消息: %s", string(msg.Body))

	if string(msg.Body) != body {
		t.Errorf("消息内容不匹配: 期望 %s, 实际 %s", body, string(msg.Body))
	}
}

// TestRabbitMQExchange 测试交换机
func TestRabbitMQExchange(t *testing.T) {
	conn, err := amqp.Dial(rabbitmqURL(t))
	if err != nil {
		t.Fatalf("无法连接到 RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("无法创建 Channel: %v", err)
	}
	defer ch.Close()

	// 声明交换机
	exName := uniqueName(t, "test_exchange")
	err = ch.ExchangeDeclare(
		exName,   // name
		"fanout", // type
		false,    // durable
		false,    // auto-deleted
		false,    // internal
		false,    // no-wait
		nil,      // arguments
	)
	if err != nil {
		t.Fatalf("无法声明交换机: %v", err)
	}

	t.Logf("成功声明交换机: %s", exName)
}

// TestRabbitMQQueueBind 测试队列绑定
func TestRabbitMQQueueBind(t *testing.T) {
	conn, err := amqp.Dial(rabbitmqURL(t))
	if err != nil {
		t.Fatalf("无法连接到 RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("无法创建 Channel: %v", err)
	}
	defer ch.Close()

	// 声明交换机
	exName := uniqueName(t, "test_exchange")
	err = ch.ExchangeDeclare(exName, "direct", false, false, false, false, nil)
	if err != nil {
		t.Fatalf("无法声明交换机: %v", err)
	}

	// 声明队列
	qName := uniqueName(t, "test_queue")
	q, err := ch.QueueDeclare(qName, false, false, false, false, nil)
	if err != nil {
		t.Fatalf("无法声明队列: %v", err)
	}

	// 绑定队列到交换机
	err = ch.QueueBind(
		q.Name,     // queue name
		"test_key", // routing key
		exName,     // exchange
		false,
		nil,
	)
	if err != nil {
		t.Fatalf("无法绑定队列: %v", err)
	}

	t.Logf("成功绑定队列 %s 到交换机 %s", q.Name, exName)
}

// TestRabbitMQMultipleMessages 测试批量发送和接收消息
func TestRabbitMQMultipleMessages(t *testing.T) {
	conn, err := amqp.Dial(rabbitmqURL(t))
	if err != nil {
		t.Fatalf("无法连接到 RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		t.Fatalf("无法创建 Channel: %v", err)
	}
	defer ch.Close()

	qName := uniqueName(t, "test_queue")
	q, err := ch.QueueDeclare(qName, false, false, false, false, nil)
	if err != nil {
		t.Fatalf("无法声明队列: %v", err)
	}

	// 发送多条消息
	messageCount := 5
	for i := 0; i < messageCount; i++ {
		body := []byte("消息 " + string(rune(i+'0')))
		err = ch.Publish("", q.Name, false, false, amqp.Publishing{
			ContentType: "text/plain",
			Body:        body,
		})
		if err != nil {
			t.Fatalf("无法发布消息 %d: %v", i, err)
		}
		log.Printf("发送消息: %s", body)
	}

	t.Logf("成功发送 %d 条消息", messageCount)
}
