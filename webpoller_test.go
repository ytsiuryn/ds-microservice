package microservice

import (
	"testing"
	"time"
)

func TestOneURL(t *testing.T) {
	poller := NewWebPoller(time.Second)
	go poller.Start()
	for _, url := range []string{"http://golang.org/", "https://golang.org/"} {
		poller.Add(url, map[string]string{})
		resource := <-poller.Completed
		if resource.Err != nil {
			t.Fatal(resource.Err)
		}
	}
}
