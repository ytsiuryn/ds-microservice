package microservice

import (
	"runtime/debug"

	log "github.com/sirupsen/logrus" // FIXME: пока неизменяем
)

// RequestRepresenter обрабатывает для журналирования входные запросы.
type RequestRepresenter interface {
	LogRequest(request RequestParser)
	Log() *DefaultLog
}

// BaseLogRepresenter базовый объект логгирования.
// Обрабатывает только наименование команд запросов.
type BaseLogRepresenter struct {
	*DefaultLog
}

// LogLevel определяет уровень журналирования исходя из того является ли система запущенной
// в Product-режиме.
func NewBaseLogRepresenter(logger *DefaultLog) *BaseLogRepresenter {
	repr := &BaseLogRepresenter{logger}
	return repr
}

func (repr *BaseLogRepresenter) Log() *DefaultLog {
	return repr.Log()
}

// Отображение сведений о выполняемом запросе.
func (repr *BaseLogRepresenter) LogRequest(requestParser RequestParser) {
	repr.Debug(requestParser.Cmd() + "()")
}

type DefaultLog struct {
	Log *log.Logger
}

// NewDefaultLog возвращает умолчательный объект журналирования.
func NewDefaultLog() *DefaultLog {
	return &DefaultLog{Log: log.New()}
}

// Info представляет короткую ссылку на одноименный метод объекта журналирования.
// FIXME: оформлять с WithField и отображением поля контекста.
func (log *DefaultLog) Info(args ...interface{}) {
	log.Log.Info(args...)
}

// Debug представляет короткую ссылку на одноименный метод объекта журналирования.
// FIXME: оформлять с WithField и отображением поля контекста.
func (log *DefaultLog) Debug(args ...interface{}) {
	log.Log.Debug(args...)
}

// FailOnError выводит сообщение об ошибке в лог и прекращает работу микросервиса.
func (log *DefaultLog) FailOnError(err error, context string) {
	if err != nil {
		debug.PrintStack()
		log.Log.WithField("context", context).Fatal(err)
	}
}

// LogOnError выводит сообщение об ошибке в лог.
func (log *DefaultLog) LogOnError(err error, context string) {
	if err != nil {
		debug.PrintStack()
		log.Log.WithField("context", context).Error(err)
	}
}
