package microservice

import (
	"runtime/debug"

	log "github.com/sirupsen/logrus"
)

// FailOnError печатает сообщение об ошибке в лог и останавливает выполнение процесса.
func FailOnError(err error, context string) {
	if err != nil {
		debug.PrintStack()
		log.WithField("context", context).Fatal(err)
	}
}

// LogOnError печатает сообщение об ошибке без указания контекста.
// В случае отсутствия ошибки не делает ничего.
func (s *Service) LogOnError(err error) {
	if err != nil {
		log.Error(err)
	}
}

// LogOnErrorWithContext печатает сообщение об ошибке в некотором контексте.
// В случае отсутствия ошибки не делает ничего.
func (s *Service) LogOnErrorWithContext(err error, context string) {
	if err != nil {
		log.WithField("context", context).Error(err)
	}
}

// LogOnErrorWithContextAndStack печатает сообщение об ошибке в некотором контексте
// и стек вызовов.
// В случае отсутствия ошибки не делает ничего.
func (s *Service) LogOnErrorWithContextAndStack(err error, context string) {
	if err != nil {
		debug.PrintStack()
		log.WithField("context", context).Error(err)
	}
}
