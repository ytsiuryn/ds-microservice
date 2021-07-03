package microservice

import (
	"io/ioutil"
	"net/http"
	"time"
)

// Описывает входной URL и результаты его обработки.
type WebResource struct {
	URL        string
	InHeaders  map[string]string
	OutHeaders map[string][]string
	Data       []byte
	Err        error
}

type WebPoller struct {
	ticker    *time.Ticker
	pending   chan *WebResource
	Completed chan *WebResource
}

// NewWebPoller формирует новый объект WebPoller.
func NewWebPoller(pollingInterval time.Duration) *WebPoller {
	return &WebPoller{
		ticker:    time.NewTicker(pollingInterval),
		pending:   make(chan *WebResource),
		Completed: make(chan *WebResource)}
}

// SetPollingInterval динамически изменяет частоту опроса.
func (wp *WebPoller) SetPollingInterval(interval time.Duration) {
	wp.ticker.Reset(interval)
}

// Add добавляет в очередь обработки новый URL.
func (wp *WebPoller) Add(url string, headers map[string]string) {
	wp.pending <- &WebResource{URL: url, InHeaders: headers}
}

// Start запускает цикл обработки запросов.
func (wp *WebPoller) Start() {
	tickerChannel := wp.ticker.C
	go func() {
		for {
			select {
			case <-tickerChannel:
				resource := <-wp.pending
				loadResource(resource)
				wp.Completed <- resource
			}
		}
	}()
}

func loadResource(resource *WebResource) {
	client := http.Client{}
	req, err := http.NewRequest("GET", resource.URL, nil)
	if err != nil {
		resource.Err = err
		return
	}

	for k, v := range resource.InHeaders {
		req.Header.Add(k, v)
	}

	response, err := client.Do(req)
	if err != nil {
		resource.Err = err
		return
	}
	defer response.Body.Close()

	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		resource.Err = err
		return
	}
	resource.Data = data
	resource.OutHeaders = response.Header
}
