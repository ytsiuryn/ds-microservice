// Модуль периодического опроса http ресурса.
//
// Функционал не будет работать корректно, если http-сервер
// не обрабатывает запросы последовательно.

// TODO: добавить в данные ресурса ID запроса (см. CreateCmdRequest()) и изменить структуру
// Completed для произвольной обработки ответов:
// клиент в начале формирования запроса получает Cmd Ref и затем опрашивает Completed  в поиске
// подходящего ответа

package microservice

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
)

// WebResource описывает входной URL и результаты его обработки.
type WebResource struct {
	URL       string
	Method    string
	InHeaders map[string]string
	Response  *http.Response
	Err       error
}

// WebPoller организует поток данных для запросов внешних ресурсов.
type WebPoller struct {
	ticker    *time.Ticker
	pending   chan *WebResource
	Completed chan *WebResource
	Log       *log.Logger
}

// NewWebPoller формирует новый объект WebPoller.
func NewWebPoller(interval time.Duration) *WebPoller {
	return &WebPoller{
		ticker:    time.NewTicker(interval),
		pending:   make(chan *WebResource),
		Completed: make(chan *WebResource)}
}

// SetPollingInterval динамически изменяет частоту опроса.
func (wp *WebPoller) SetPollingInterval(interval time.Duration) {
	wp.ticker.Reset(interval)
}

// Add добавляет в очередь обработки новый http-ресурс.
func (wp *WebPoller) Add(url, method string, headers map[string]string) {
	wp.Log.Debug(url)
	wp.pending <- &WebResource{URL: url, InHeaders: headers}
}

// Head дожидается выполнения команды "HEAD" с возвратом объекта http.
func (wp *WebPoller) Head(url string, headers map[string]string) *WebResource {
	wp.Add(url, "HEAD", headers)
	return <-wp.Completed
}

// Get выполняет команду "GET" возвращает объект WebResource с объектом ответа http.Response.
func (wp *WebPoller) Get(url string, headers map[string]string) *WebResource {
	wp.Add(url, "GET", headers)
	return <-wp.Completed
}

// Load возвращает содержимое тела http ресурса.
func (wp *WebPoller) Load(url string, headers map[string]string) ([]byte, error) {
	wp.Add(url, "GET", headers)

	resource := <-wp.Completed
	if resource.Response == nil {
		return nil, errors.New("no Internet connection")
	}
	defer resource.Response.Body.Close()

	data, err := ioutil.ReadAll(resource.Response.Body)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// DecodeJSON загружает http ресурс и декодирует JSON данные ресурса.
func (wp *WebPoller) DecodeJSON(url string, headers map[string]string, out interface{}) error {
	data, err := wp.Load(url, headers)
	if err != nil {
		if wp.Log != nil {
			wp.Log.Error(string(data))
		}
		return err
	}
	return json.Unmarshal(data, &out)
}

// Start запускает цикл обработки запросов.
func (wp *WebPoller) Start() {
	tickerChannel := wp.ticker.C
	go func() {
		for {
			select {
			case <-tickerChannel:
				select {
				case resource := <-wp.pending:
					loadResource(resource)
					wp.Completed <- resource
				default:
				}
			}
		}
	}()
}

// Синхронный запрос к http-ресурсу.
func loadResource(resource *WebResource) {
	client := http.Client{}
	req, err := http.NewRequest(resource.Method, resource.URL, nil)
	if err != nil {
		resource.Err = err
		return
	}

	if resource.InHeaders != nil {
		for k, v := range resource.InHeaders {
			req.Header.Add(k, v)
		}
	}

	if response, err := client.Do(req); err != nil {
		resource.Err = err
	} else {
		resource.Response = response
	}
}
