package microservice

import (
	"encoding/json"

	"github.com/streadway/amqp"
)

// Request - интерфейс для объектов, разбирающих входные запросы от клиентов.
type Request interface {
	Parse(d *amqp.Delivery) error
}

// BaseRequest - базовый запрос для выполнения команд, не требующих аргументов (только
// наименование команды).
type BaseRequest struct {
	Cmd string
}

// ParseRequest проверяет и извлекает данные запроса.
func (br *BaseRequest) Parse(d *amqp.Delivery) error {
	if err := json.Unmarshal(d.Body, br); err != nil {
		return err
	}
	return nil
}
