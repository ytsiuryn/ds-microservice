package microservice

import (
	"encoding/json"

	"github.com/streadway/amqp"
)

// Requester - интерфейс для объектов, разбирающих входные запросы от клиентов.
type Requester interface {
	Parse(d *amqp.Delivery) error
	// Request() Request
	Cmd() string
}

// BaseRequest - базовый запрос для выполнения команд, не требующих аргументов (только
// наименование команды).
type BaseRequest struct {
	Cmd string
}

// BaseRequester - базовый парсер, умеющий только извлекать команду запроса.
type BaseRequester struct {
	request BaseRequest
}

// ParseRequest проверяет и извлекает данные запроса.
func (brp *BaseRequester) Parse(d *amqp.Delivery) error {
	if err := json.Unmarshal(d.Body, &brp.request); err != nil {
		return err
	}
	return nil
}

// Cmd выводит наименование команды для последнего разобранного запроса.
func (brp *BaseRequester) Cmd() string {
	return brp.request.Cmd
}
