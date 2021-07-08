package microservice

import (
	"errors"
	"runtime/debug"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// Структура хранения данных для ответа с ошибкой.
type ErrorResponse struct {
	Error   string `json:"error"`
	Context string `json:"context"`
}

// Service хранит состояние микросервиса.
type Service struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	Log  *log.Logger
	Name string
}

// NewService возвращает новую копию объекта Service.
func NewService(srvName string) *Service {
	return &Service{Log: log.New()}
}

// ConnectToMessageBroker подключает микросервис под именем `name` к брокеру сообщений.
// Дополнительно go-канал обмена сообщений с брокером передается диспетчеру для обработки
// последующих запросов.
func (s *Service) ConnectToMessageBroker(connstr string) <-chan amqp.Delivery {
	conn, err := amqp.Dial(connstr)
	FailOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	FailOnError(err, "Failed to open a channel")

	q, err := ch.QueueDeclare(
		s.Name, // name
		false,  // durable
		false,  // delete when unused
		false,  // exclusive
		false,  // no-wait
		nil,    // arguments
	)
	FailOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	FailOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto ack
		false,  // exclusive
		false,  // no local
		false,  // no wait
		nil,    // args
	)
	FailOnError(err, "Failed to register a consumer")

	s.conn = conn
	s.ch = ch

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

// ErrorResult отправляет клиенту ответ с информацией об ошибке.
func (s *Service) AnswerWithError(delivery *amqp.Delivery, e error, context string) {
	s.LogOnError(e, context)
	json := []byte("{\"error\": \"" + e.Error() + "\", \"context\": \"" + context + "\"}")
	s.Answer(delivery, json)
}

// Answer отправляет клиенту ответ `result` в JSON формате в соответствии с идентификатором
// запроса CorrelationId в параметре delivery.
func (s *Service) Answer(delivery *amqp.Delivery, result []byte) {
	if err := s.ch.Publish(
		"",
		delivery.ReplyTo,
		false,
		false,
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: delivery.CorrelationId,
			Body:          result,
		}); err != nil {
		s.AnswerWithError(delivery, err, "Answer's publishing error")
		return
	}
	delivery.Ack(false)
}

// Ping сигнализирует о работоспособности микросервиса с пустым ответом.
func (s *Service) Ping(delivery *amqp.Delivery) {
	s.Answer(delivery, []byte{})
}

// LogOnError print out an error message into log
func (s *Service) LogOnError(err error, context string) {
	if err != nil {
		debug.PrintStack()
		log.WithField("context", context).Error(err)
	}
}
