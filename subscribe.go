// Модуль реализации схемы публикатор/подписчик с использованием RabbitMQ.

package microservice

import (
	"github.com/streadway/amqp"
)

// Publisher ..
type Publisher struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	exchType string
	exchName string
}

// Subscriber ..
type Subscriber struct {
	conn     *amqp.Connection
	ch       *amqp.Channel
	q        amqp.Queue
	exchType string
	exchName string
}

// NewPublisher создает объект издателя, подключенный к Exchange типа `fanout`.
func NewPublisher(exchName string) *Publisher {
	return &Publisher{
		exchType: "fanout",
		exchName: exchName,
	}
}

// NewSubscriber создает объект подписчика, подключенный к Exchange типа `fanout`.
func NewSubscriber(exchName string) *Subscriber {
	return &Subscriber{
		exchType: "fanout",
		exchName: exchName,
	}
}

// Connect выполняет соединение с брокером и инициалирует Exchange.
// В случае ошибки процесс завершает свою работу.
func (pub *Publisher) Connect(connStr string) {
	var err error
	pub.conn, err = amqp.Dial(connStr)
	FailOnError(err, "Failed to connect to RabbitMQ")
	pub.ch, err = pub.conn.Channel()
	FailOnError(err, "Failed to open a channel")
	err = pub.ch.ExchangeDeclare(
		pub.exchName,
		pub.exchType,
		true,
		false,
		false,
		false,
		nil,
	)
	FailOnError(err, "Failed to declare an exchange")
}

// Close освобождает ресурсы Emitter.
func (pub *Publisher) Close() {
	pub.ch.Close()
	pub.conn.Close()
}

// Emit отправляет сообщение подписчикам.
// contentType = "text/plain"
func (pub *Publisher) Emit(contentType string, data []byte) error {
	err := pub.ch.Publish(
		pub.exchName,
		"",
		false,
		false,
		amqp.Publishing{
			ContentType: contentType,
			Body:        data,
		})
	if err != nil {
		return err
	}
	return nil
}

// Connect выполняет соединение с брокером.
// Также настраивает Exchange, создает новую очередь и связывает ее с Exchange.
// В случае ошибки процесс останавливается.
func (sub *Subscriber) Connect(connStr string) {
	var err error
	sub.conn, err = amqp.Dial(connStr)
	FailOnError(err, "Failed to connect to RabbitMQ")
	sub.ch, err = sub.conn.Channel()
	FailOnError(err, "Failed to open a channel")
	err = sub.ch.ExchangeDeclare(
		sub.exchName,
		sub.exchType,
		true,
		false,
		false,
		false,
		nil,
	)
	FailOnError(err, "Failed to declare an exchange")
	sub.q, err = sub.ch.QueueDeclare(
		"",
		false,
		false,
		true,
		false,
		nil,
	)
	FailOnError(err, "Failed to declare a queue")
	err = sub.ch.QueueBind(
		sub.q.Name,
		"",
		sub.exchName,
		false,
		nil,
	)
	FailOnError(err, "Failed to bind a queue")
}

// Close освобождает ресурсы подписчика по завершении его работы.
func (sub *Subscriber) Close() {
	sub.ch.Close()
	sub.conn.Close()
}

// Receive в цикле принимает сообщения от издателя и пересылает их в предоставленный выходной канал.
// В случае ошибки цикл приема сообщений прерывается.
func (sub *Subscriber) Receive(out chan<- amqp.Delivery) {
	msgs, err := sub.ch.Consume(
		sub.q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	FailOnError(err, "Failed to register a consumer")
	forever := make(chan bool)
	go func() {
		for d := range msgs {
			out <- d
		}
	}()
	<-forever
}

// ReceiveOnce выполняет прием одного сообщения.
// Применяется для тестовых целей.
func (sub *Subscriber) ReceiveOnce() amqp.Delivery {
	msgs, err := sub.ch.Consume(
		sub.q.Name,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	FailOnError(err, "Failed to register a consumer")
	return <-msgs
}
