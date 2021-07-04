package microservice

import (
	"encoding/json"

	"github.com/streadway/amqp"
)

// RequestParser - интерфейс для объектов, разбирающих входные запросы от клиентов.
type RequestParser interface {
	Parse(d *amqp.Delivery) error
	// Request() Request
	Cmd() string
}

// BaseRequest - базовый запрос для выполнения команд, не требующих аргументов (только
// наименование команды).
type BaseRequest struct {
	Cmd string
}

// BaseRequestParser - базовый парсер, умеющий только извлекать команду запроса.
type BaseRequestParser struct {
	request BaseRequest
}

// ParseRequest проверяет и извлекает данные запроса.
func (brp *BaseRequestParser) Parse(d *amqp.Delivery) error {
	if err := json.Unmarshal(d.Body, &brp.request); err != nil {
		return err
	}
	return nil
}

// Cmd выводит наименование команды для последнего разобранного запроса.
func (brp *BaseRequestParser) Cmd() string {
	return brp.request.Cmd
}
