package microservice

import (
	"testing"
	"time"
)

func TestWebPollerLoad(t *testing.T) {
	poller := NewWebPoller(time.Second)
	go poller.Start()
	for _, url := range []string{"http://golang.org/", "https://golang.org/"} {
		_, err := poller.Load(url, map[string]string{})
		if err != nil {
			t.Fatal(err)
		}
	}
}
