// Главный модуль, реализующий основную функциональность для организации микросервисов.
//
// Обработка команд выполняется в RPC-режиме.
// Команды представлены в JSON-формате (см. rpc_client.go).
//
// Дополнительно могут быть запущены схемы обмена данными "публикатор/подписчик" для
// широкополосного оповещения и синхронизированный опрос http-ресурсов

package microservice

import (
	"errors"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// DefaultRabbitMQConnStr - умолчательный адрес локального подключения к брокеру сообщений.
const DefaultRabbitMQConnStr = "amqp://guest:guest@localhost:5672/"

// ErrorResponse хранит данные для ответа с ошибкой.
// Используется для тестовых целей.
type ErrorResponse struct {
	Error   string `json:"error,omitempty"`
	Context string `json:"context,omitempty"`
}

// Service хранит состояние микросервиса.
type Service struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	q    amqp.Queue
	Log  *log.Logger
	Name string
}

// NewService возвращает новую копию объекта Service.
func NewService(srvName string) *Service {
	return &Service{Log: log.New(), Name: srvName}
}

// ConnectToMessageBroker подключает микросервис под именем `name` к брокеру сообщений.
// Дополнительно go-канал обмена сообщений с брокером передается диспетчеру для обработки
// последующих запросов.
func (s *Service) ConnectToMessageBroker(connstr string) <-chan amqp.Delivery {
	var err error

	s.conn, err = amqp.Dial(connstr)
	FailOnError(err, "Failed to connect to RabbitMQ")

	s.ch, err = s.conn.Channel()
	FailOnError(err, "Failed to open a channel")

	s.q, err = s.ch.QueueDeclare(
		s.Name, // name
		false,  // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	FailOnError(err, "Failed to declare a queue")

	err = s.ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	FailOnError(err, "Failed to set QoS")

	msgs, err := s.ch.Consume(
		s.q.Name, // queue
		"",       // consumer
		false,    // auto ack
		false,    // exclusive
		false,    // no local
		false,    // no wait
		nil,      // args
	)
	FailOnError(err, "Failed to register a consumer")

	return msgs
}

// Cleanup освобождает ресурсы и выводит сообщение о завершении работы сервиса.
func (s *Service) Cleanup() {
	s.ch.Close()
	s.conn.Close()
	s.Log.Infoln("stopped")
}

// RunCmd вызывает командам  запроса методы сервиса и возвращает результат клиенту.
func (s *Service) RunCmd(cmd string, delivery *amqp.Delivery) {
	switch cmd {
	case "ping":
		go s.Ping(delivery)
	default:
		go s.AnswerWithError(
			delivery,
			errors.New("Unknown command: "+cmd),
			"Message dispatcher")
	}
}

// AnswerWithError отправляет клиенту ответ с информацией об ошибке.
func (s *Service) AnswerWithError(delivery *amqp.Delivery, e error, context string) {
	s.LogOnErrorWithContext(e, context)
	json := []byte("{\"error\": \"" + e.Error() + "\", \"context\": \"" + context + "\"}")
	s.Answer(delivery, json)
}

// Answer отправляет клиенту ответ `result` в JSON формате в соответствии с идентификатором
// запроса CorrelationId в параметре delivery.
// В случае ошибки отправки работа сервиса прекращается.
func (s *Service) Answer(delivery *amqp.Delivery, result []byte) {
	err := s.ch.Publish(
		"",
		delivery.ReplyTo,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: delivery.CorrelationId,
			Body:          result,
		})
	FailOnError(err, "Answer's publishing error")

	FailOnError(delivery.Ack(false), "Acknowledge error")
}

// Ping сигнализирует о работоспособности микросервиса с пустым ответом.
func (s *Service) Ping(delivery *amqp.Delivery) {
	s.Answer(delivery, []byte{})
}
