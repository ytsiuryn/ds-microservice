package microservice

import (
	"runtime/debug"

	log "github.com/sirupsen/logrus"
)

// ServiceLogRepresenter описывает объект логгирования, который интерпретирует
// входную, выходную информацию и сведения об ошибке.
type ServiceLogRepresenter interface {
	LogRequest(request Requester)
	Logger() *log.Logger
	FailOnError(err error, context string)
	LogOnError(err error, context string)
}

// BaseLogRepresenter базовый объект логгирования.
// Обрабатывает только наименование команд запросов.
type BaseLogRepresenter struct {
	logger *log.Logger
	level  log.Level
}

// LogLevel определяет уровень журналирования исходя из того является ли система запущенной
// в Product-режиме.
func NewBaseLogRepresenter(isProduct bool) *BaseLogRepresenter {
	ret := &BaseLogRepresenter{logger: log.New()}
	if isProduct {
		ret.level = log.InfoLevel
	}
	ret.level = log.DebugLevel
	return ret
}

// Logger возвращает ссылку на объект логирования.
func (repr *BaseLogRepresenter) Logger() *log.Logger {
	return repr.logger
}

// Отображение сведений о выполняемом запросе.
func (repr *BaseLogRepresenter) LogRequest(requestParser Requester) {
	repr.logger.Debug(requestParser.Cmd() + "()")
}

// FailOnError выводит сообщение об ошибке в лог и прекращает работу микросервиса.
func (repr *BaseLogRepresenter) FailOnError(err error, context string) {
	if err != nil {
		debug.PrintStack()
		repr.logger.WithField("context", context).Fatal(err)
	}
}

// LogOnError выводит сообщение об ошибке в лог.
func (repr *BaseLogRepresenter) LogOnError(err error, context string) {
	if err != nil {
		debug.PrintStack()
		repr.logger.WithField("context", context).Error(err)
	}
}
