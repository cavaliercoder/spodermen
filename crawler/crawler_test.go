package crawler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"
	"time"
)

func TestOverflow(t *testing.T) {
	var i int64
	var ts *httptest.Server
	ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a := atomic.AddInt64(&i, 1)
		b := atomic.AddInt64(&i, 1)
		fmt.Fprintf(w, `<html>
  <head>
    <title>Test</title>
  </head>
  <body>
		<h1>Test</h1>
		<p><a href="%s/%v">Link %v</a></p>
		<p><a href="%s/%v">Link %v</a></p>
  </body>
</html>`, ts.URL, a, a, ts.URL, b, b)
	}))
	defer ts.Close()

	req, _ := NewRequest(ts.URL, true)
	ctx, _ := context.WithTimeout(context.Background(), time.Minute)
	q := NewQueue(8, ctx)
	q.Enqueue(req)
	u, _ := url.Parse(ts.URL)
	c, _ := New(FollowHosts(u.Host))
	stats := c.Crawl(q, 4)
	t.Log(stats.JSON())
}
