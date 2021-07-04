package microservice

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/streadway/amqp"
)

// Dispatcher принимает управление микросервисом на себя и обрабатывает поступающие запросы.
type Dispatcher interface {
	SetRequestRepresenter(repr RequestRepresenter)
	Dispatch()
}

// BaseDispatcher - базовый диспетчер микросервисов.
type BaseDispatcher struct {
	msgs      <-chan amqp.Delivery
	service   Microservice
	requester RequestParser
	repr      RequestRepresenter
}

// NewBaseDispatcher создает объект диспетчера микросервиса.
func NewBaseDispatcher(msgs <-chan amqp.Delivery, service Microservice) *BaseDispatcher {
	return &BaseDispatcher{
		msgs:      msgs,
		service:   service,
		requester: &BaseRequestParser{}}
}

func (d *BaseDispatcher) SetRequestRepresenter(repr RequestRepresenter) {
	d.repr = repr
}

// Dispatch выполняет цикл обработки взодящих запросов.
// Также контролирует сигнал завершения цикла и последующего освобождения ресурсов микросервиса.
func (d *BaseDispatcher) Dispatch() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		for delivery := range d.msgs {
			err := d.requester.Parse(&delivery)
			if err != nil {
				d.service.AnswerWithError(&delivery, err, "Message dispatcher")
				continue
			}
			d.repr.LogRequest(d.requester)
			d.service.RunCmd(d.requester, &delivery)
		}
	}()
	d.repr.Log().Info("Awaiting RPC requests")
	<-c
}
