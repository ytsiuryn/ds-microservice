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

// Microservice - базовый интерфейс для реализации микросервисов.
type Microservice interface {
	ConnectToMessageBroker(connstr string)
	RunCmd(req RequestParser, delivery *amqp.Delivery)
	Answer(delivery *amqp.Delivery, result []byte)
	AnswerWithError(delivery *amqp.Delivery, e error, context string)
	Cleanup()
}

// ServiceInfo хранит общие сведения о микросервисе.
type ServiceInfo struct {
	Subsystem, Name, Description, Date string
}

// Service хранит состояние микросервиса.
type Service struct {
	conn       *amqp.Connection
	ch         *amqp.Channel
	msgs       <-chan amqp.Delivery
	dispatcher Dispatcher
	info       ServiceInfo
	poller     *WebPoller
	Log        *DefaultLog
}

// NewService возвращает новую копию объекта Service.
func NewService(srvName string) *Service {
	return &Service{
		info: ServiceInfo{Name: srvName},
		Log:  NewDefaultLog()}
}

// ConnectToMessageBroker подключает микросервис под именем `name` к брокеру сообщений.
// Дополнительно go-канал обмена сообщений с брокером передается диспетчеру для обработки
// последующих запросов.
func (s *Service) ConnectToMessageBroker(connstr string) {
	conn, err := amqp.Dial(connstr)
	s.Log.FailOnError(err, "Failed to connect to RabbitMQ")

	ch, err := conn.Channel()
	s.Log.FailOnError(err, "Failed to open a channel")

	q, err := ch.QueueDeclare(
		s.info.Name, // name
		false,       // durable
		false,       // delete when unused
		false,       // exclusive
		false,       // no-wait
		nil,         // arguments
	)
	s.Log.FailOnError(err, "Failed to declare a queue")

	err = ch.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	)
	s.Log.FailOnError(err, "Failed to set QoS")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto ack
		false,  // exclusive
		false,  // no local
		false,  // no wait
		nil,    // args
	)
	s.Log.FailOnError(err, "Failed to register a consumer")

	s.conn = conn
	s.ch = ch
	s.msgs = msgs
}

// Dispatch запускает цикл опроса входящих запросов.
func (s *Service) Dispatch() {
	dispatcher := NewBaseDispatcher(s.msgs, s)
	dispatcher.SetRequestRepresenter(NewBaseLogRepresenter(s.Log))
	dispatcher.Dispatch()
}

// SetInfo устанавливает описание микросервиса.
func (s *Service) SetInfo(info ServiceInfo) {
	s.info = info
}

// StartWebPoller иниуиализирует и запускает цикл обработки запросов Web ресурсов с заданной
// цикличностью.
func (s *Service) StartPoller(interval time.Duration) {
	s.poller = NewWebPoller(interval)
	s.poller.Start()
}

// Msgs возвращает канал поставки входных сообщений от клиента.
func (s *Service) Msgs() <-chan amqp.Delivery {
	return s.msgs
}

// Poller возвращает ссылку на объект лимитированного по частоту обращений объекта опроса.
func (s *Service) Poller() *WebPoller {
	return s.poller
}

// ReadConfig читает содержимое файла настройки сервиса в выходную структуру.
// TODO: вынести за рамки сервиса.
func (s *Service) ReadConfig(optFile string, conf interface{}) {
	data, err := ioutil.ReadFile(optFile)
	s.Log.FailOnError(err, "Config file")
	s.Log.FailOnError(yaml.Unmarshal(data, conf), "Config file")
}

// Close закрывает соединения с RabbitMq.
func (s *Service) Close() {
	s.ch.Close()
	s.conn.Close()
}

// Cleanup освобождает ресурсы и выводит сообщение о завершении работы сервиса.
func (s *Service) Cleanup() {
	s.Close()
	s.Log.Info("\nstopped")
}

// RunCmd вызывает командам  запроса методы сервиса и возвращает результат клиенту.
func (s *Service) RunCmd(req RequestParser, delivery *amqp.Delivery) {
	switch req.Cmd() {
	case "ping":
		go s.Ping(delivery)
	case "info":
		go s.Info(delivery)
	default:
		go s.AnswerWithError(
			delivery,
			errors.New("Unknown command: "+req.Cmd()),
			"Message dispatcher")
	}
}

// ErrorResult отправляет клиенту ответ с информацией об ошибке.
func (s *Service) AnswerWithError(delivery *amqp.Delivery, e error, context string) {
	s.Log.LogOnError(e, context)
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
