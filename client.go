package microservice

import (
	"encoding/json"

	"github.com/gofrs/uuid"
	"github.com/streadway/amqp"
)

// Структура хранения данных для ответа с ошибкой.
type ErrorResponse struct {
	Error   string `json:"error"`
	Context string `json:"context"`
}

// RPCClient хранит состояние клиента микросервиса
type RPCClient struct {
	conn   *amqp.Connection
	ch     *amqp.Channel
	msgs   <-chan amqp.Delivery
	q      amqp.Queue
	logger ServiceLogRepresenter
}

// NewRPCClient создает новый объект клиента микросервиса.
func NewRPCClient(logger ServiceLogRepresenter) *RPCClient {
	var err error
	cl := &RPCClient{logger: logger}
	cl.conn, err = amqp.Dial("amqp://guest:guest@localhost:5672/")
	cl.logger.FailOnError(err, "Failed to connect to RabbitMQ")

	cl.ch, err = cl.conn.Channel()
	cl.logger.FailOnError(err, "Failed to open a channel")

	cl.q, err = cl.ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // noWait
		nil,   // arguments
	)
	cl.logger.FailOnError(err, "Failed to declare a queue")

	cl.msgs, err = cl.ch.Consume(
		cl.q.Name, // queue
		"",        // consumer
		true,      // auto-ack
		false,     // exclusive
		false,     // no-local
		false,     // no-wait
		nil,       // args
	)
	cl.logger.FailOnError(err, "Failed to register a consumer")

	return cl
}

// Request выполняет запрос к микросервису по имени `srvName`.
// Запрос квитируется уникальным идентификатором corrID.
// Поле `args` содержит JSON представление запроса.
func (cl *RPCClient) Request(srvName, corrID string, args []byte) {
	err := cl.ch.Publish(
		"",      // exchange
		srvName, // routing key
		false,   // mandatory
		false,   // immediate
		amqp.Publishing{
			ContentType:   "application/json",
			CorrelationId: corrID,
			ReplyTo:       cl.q.Name,
			Body:          args,
		})
	cl.logger.FailOnError(err, "Failed to publish a message")
}

// Result блокирует ход исполнения до момента ответа микросервиса на запрос и
// возвращает сам ответ в виде JSON.
func (cl *RPCClient) Result(correlationID string) []byte {
	for d := range cl.msgs {
		if correlationID == d.CorrelationId {
			return d.Body
		}
	}
	return nil
}

// Close освобождает ресурсы клиента при его закрытии.
func (cl *RPCClient) Close() {
	cl.conn.Close()
	cl.ch.Close()
}

// CreateCmdRequest формирует данные для запроса команды `cmd` в виде CorrelationID и тела
// JSON.
func CreateCmdRequest(cmd string) (string, []byte, error) {
	correlationID, _ := uuid.NewV4()
	request := BaseRequest{cmd}
	reqData, err := json.Marshal(&request)
	if err != nil {
		return "", nil, err
	}
	return correlationID.String(), reqData, nil
}

// ParseErrorAnswer разбирает ответ с описанием ошибки.
func ParseErrorAnswer(data []byte) (*ErrorResponse, error) {
	ret := &ErrorResponse{}
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}
	return ret, nil
}
