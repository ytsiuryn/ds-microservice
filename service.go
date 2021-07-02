package microservice

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/streadway/amqp"
	"gopkg.in/yaml.v3"
)

// ServiceInfo хранит общие сведения о микросервисе.
type ServiceInfo struct {
	Subsystem, Name, Description, Date string
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

// Microservice - базовый интерфейс для реализации микросервисов.
type Microservice interface {
	ConnectToMessageBroker(connstr, name string)
	RunCmd(req Requester, delivery *amqp.Delivery)
	Answer(delivery *amqp.Delivery, result []byte)
	ErrorResult(delivery *amqp.Delivery, e error, context string)
	Cleanup()
}

// Service хранит состояние микросервиса.
type Service struct {
	Conn       *amqp.Connection
	Ch         *amqp.Channel
	Idle       bool
	logger     ServiceLogRepresenter
	dispatcher Dispatcher
	info       ServiceInfo
}

// NewService возвращает новую копию объекта Service.
func NewService(logger ServiceLogRepresenter) *Service {
	srv := &Service{
		logger: logger,
		info: ServiceInfo{
			Subsystem:   "-",
			Name:        "base_service",
			Description: "base microservice"}}
	srv.logger.FailOnError(srv.info.InitializeExecModTime(), "Service initialization")
	return srv
}

// ConnectToMessageBroker подключает микросервис к брокеру сообщений.
// Дополнительно go-канал обмена сообщений с брокером передается диспетчеру для обработки
// последующих запросов.
func (s *Service) ConnectToMessageBroker(connstr, name string) {
	conn, err := amqp.Dial(connstr)
	s.logger.FailOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	s.logger.FailOnError(err, "Failed to open a channel")

	q, err := ch.QueueDeclare(
		name,  // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	s.logger.FailOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	s.logger.FailOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto ack
		false,  // exclusive
		false,  // no local
		false,  // no wait
		nil,    // args
	)
	s.logger.FailOnError(err, "Failed to register a consumer")

	s.Conn = conn
	s.Ch = ch
	s.info.Name = name

	s.dispatcher = NewBaseDispatcher(msgs, s)
}

// SetDispatcher используется для перегрузки умолчательного диспетчера BaseDispatcher другим.
func (s *Service) SetDispatcher(dispatcher Dispatcher) {
	s.dispatcher = dispatcher
}

// SetLogRepresenter используется для перегрузки умолчательного BaseLogRepresenter другим.
func (s *Service) SetLogRepresenter(logger ServiceLogRepresenter) {
	s.logger = logger
}

// SetInfo устанавливает описание микросервиса, порожденного от базового.
func (s *Service) SetInfo(info ServiceInfo) {
	s.info = info
}

// ReadConfig читает содержимое файла настройки сервиса в выходную структуру.
func (s *Service) ReadConfig(optFile string, conf interface{}) {
	data, err := ioutil.ReadFile(optFile)
	s.logger.FailOnError(err, "Config file")
	s.logger.FailOnError(yaml.Unmarshal(data, conf), "Config file")
}

// Close закрывает соединения с RabbitMq.
func (s *Service) Close() {
	s.Ch.Close()
	s.Conn.Close()
}

// Cleanup освобождает ресурсы и выводит сообщение о завершении работы сервиса.
func (s *Service) Cleanup() {
	s.Close()
	s.logger.Info("\nstopped")
}

// RunCmd вызывает командам  запроса методы сервиса и возвращает результат клиенту.
func (s *Service) RunCmd(req Requester, delivery *amqp.Delivery) {
	switch req.Cmd() {
	case "ping":
		go s.Ping(delivery)
	case "info":
		go s.Info(delivery)
	default:
		go s.ErrorResult(
			delivery,
			errors.New("Unknown command: "+req.Cmd()),
			"Message dispatcher")
	}
}

// ErrorResult отправляет клиенту ответ с информацией об ошибке.
func (s *Service) ErrorResult(delivery *amqp.Delivery, e error, context string) {
	s.logger.LogOnError(e, context)
	json := []byte(fmt.Sprintf("{\"error\": \"%s\", \"context\": \"%s\"}", e, context))
	s.Answer(delivery, json)
}

// Answer отправляет клиенту ответ `result` в JSON формате в соответствии с идентификатором
// запроса CorrelationId в параметре delivery.
func (s *Service) Answer(delivery *amqp.Delivery, result []byte) {
	err := s.Ch.Publish(
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
		s.ErrorResult(delivery, err, "Answer's publishing error")
		return
	}
	delivery.Ack(false)
}

// Info отправляет клиенту общую информацию о микросервисе на основании данных структуры
// `ServiceInfo`.
func (s *Service) Info(delivery *amqp.Delivery) {
	json, err := json.Marshal(s.info)
	if err != nil {
		s.ErrorResult(delivery, err, fmt.Sprintf("Structure conversion error for %+v", s.info))
		return
	}
	s.Answer(delivery, json)
}

// Ping сигнализирует о работоспособности микросервиса с пустым ответом.
func (s *Service) Ping(delivery *amqp.Delivery) {
	s.Answer(delivery, []byte{})
}
