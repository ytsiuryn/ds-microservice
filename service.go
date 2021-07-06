package microservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

// ServiceInfo хранит общие сведения о микросервисе.
type ServiceInfo struct {
	Subsystem, Name, Description, Date string
}

// Service хранит состояние микросервиса.
type Service struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	info ServiceInfo
	Log  *log.Logger
}

// NewService возвращает новую копию объекта Service.
func NewService(srvName string) *Service {
	return &Service{
		info: ServiceInfo{Name: srvName}, Log: log.New()}
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
		s.info.Name, // name
		false,       // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
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

// SetInfo устанавливает описание микросервиса.
func (s *Service) SetInfo(info ServiceInfo) {
	s.info = info
}

// Cleanup освобождает ресурсы и выводит сообщение о завершении работы сервиса.
func (s *Service) Cleanup() {
	s.close()
	s.Log.Infoln("stopped")
}

func (s *Service) close() {
	s.ch.Close()
	s.conn.Close()
}

// RunCmd вызывает командам  запроса методы сервиса и возвращает результат клиенту.
func (s *Service) RunCmd(cmd string, delivery *amqp.Delivery) {
	switch cmd {
	case "ping":
		go s.Ping(delivery)
	case "info":
		go s.Info(delivery)
	default:
		go s.AnswerWithError(
			delivery,
			errors.New("Unknown command: "+cmd),
			"Message dispatcher")
	}
}

// ErrorResult отправляет клиенту ответ с информацией об ошибке.
func (s *Service) AnswerWithError(delivery *amqp.Delivery, e error, context string) {
	LogOnError(e, context)
	json := []byte(fmt.Sprintf("{\"error\": \"%s\", \"context\": \"%s\"}", e, context))
	s.Answer(delivery, json)
}

// Answer отправляет клиенту ответ `result` в JSON формате в соответствии с идентификатором
// запроса CorrelationId в параметре delivery.
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
	if err != nil {
		s.AnswerWithError(delivery, err, "Answer's publishing error")
		return
	}
	delivery.Ack(false)
}

// Info отправляет клиенту общую информацию о микросервисе на основании данных структуры
// `ServiceInfo`.
func (s *Service) Info(delivery *amqp.Delivery) {
	json, err := json.Marshal(s.info)
	if err != nil {
		s.AnswerWithError(delivery, err, fmt.Sprintf("Structure conversion error for %+v", s.info))
		return
	}
	s.Answer(delivery, json)
}

// Ping сигнализирует о работоспособности микросервиса с пустым ответом.
func (s *Service) Ping(delivery *amqp.Delivery) {
	s.Answer(delivery, []byte{})
}

// InitializeExecModTime считывает сведения о последней модификации исполняемого файла
// микросервиса и записывает их в структуру объекта.
func (i ServiceInfo) InitializeExecModTime() error {
	path, err := os.Executable()
	if err != nil {
		return errors.New("Getting microservice executable error")
	}
	info, err := os.Stat(path)
	if err != nil {
		return errors.New("Getting microservice executable stat info error")
	}
	i.Date = info.ModTime().Format(time.UnixDate)
	return nil
}
