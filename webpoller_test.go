package microservice

import (
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
)

func TestWebPollerLoad(t *testing.T) {
	poller := NewWebPoller(time.Millisecond)
	poller.Log = log.New()
	poller.Start()
	for _, url := range []string{"http://www.google.com/", "https://golang.org/"} {
		if _, err := poller.Load(url, nil); err != nil {
			t.Fatal(err)
		}
	}
}
