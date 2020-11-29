package autoupdater

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func createTestServer() *httptest.Server {
	version := 0
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%d", version)
		version++
	}))
}

func TestPoller(t *testing.T) {
	ts := createTestServer()
	url, _ := url.Parse(ts.URL)
	poller := newPoller(*url, 10*time.Millisecond)

	poller.Start()
	for i := 0; i < 10; i++ {
		if a := <-poller.changes; !bytes.Equal(a, []byte(fmt.Sprint(i))) {
			t.Errorf("Wrong polling result %s, should be %d", a, i)
		}
	}
}
