package microservice

import (
	"testing"
	"time"
)

func TestWebPollerLoad(t *testing.T) {
	poller := NewWebPoller(time.Millisecond)
	poller.Start()
	for _, url := range []string{"http://www.google.com/", "https://golang.org/"} {
		if _, err := poller.Load(url, nil); err != nil {
			t.Fatal(err)
		}
	}
}
