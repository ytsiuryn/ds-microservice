package microservice

import (
	"runtime/debug"

	log "github.com/sirupsen/logrus"
)

// ServiceLogRepresenter описывает объект логгирования, который интерпретирует
// входную, выходную информацию и сведения об ошибке.
type ServiceLogRepresenter interface {
	LogRequest(request Requester)
	FailOnError(err error, context string)
	LogOnError(err error, context string)
	Info(args ...interface{})
	Warnf(format string, args ...interface{})
}

// BaseLogRepresenter базовый объект логгирования.
// Обрабатывает только наименование команд запросов.
type BaseLogRepresenter struct {
	level log.Level
}

// LogLevel определяет уровень журналирования исходя из того является ли система запущенной
// в Product-режиме.
func NewBaseLogRepresenter(isProduct bool) *BaseLogRepresenter {
	ret := &BaseLogRepresenter{}
	if isProduct {
		ret.level = log.InfoLevel
	}
	ret.level = log.DebugLevel
	return ret
}

// Отображение сведений о выполняемом запросе.
func (logger *BaseLogRepresenter) LogRequest(requestParser Requester) {
	log.Debug(requestParser.Cmd() + "()")
}

// FailOnError выводит сообщение об ошибке в лог и прекращает работу микросервиса.
func (logger *BaseLogRepresenter) FailOnError(err error, context string) {
	if err != nil {
		debug.PrintStack()
		log.WithField("context", context).Fatal(err)
	}
}

// LogOnError выводит сообщение об ошибке в лог.
func (logger *BaseLogRepresenter) LogOnError(err error, context string) {
	if err != nil {
		debug.PrintStack()
		log.WithField("context", context).Error(err)
	}
}

// Info - хелпер для вызова аналогичного объекта логирования Info.
func (logger *BaseLogRepresenter) Info(args ...interface{}) {
	log.Info(args...)
}

// Warnf - хелпер для вызова аналогичного объекта логирования Warnf.
func (logger *BaseLogRepresenter) Warnf(format string, args ...interface{}) {
	log.Warnf(format, args...)
}
