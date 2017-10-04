package crawler

import (
	"bytes"
	"fmt"
	"text/template"
	"time"
)

var responseTemplate = func() *template.Template {
	t, err := template.New("crawl_response").
		Funcs(template.FuncMap{
			"ms": func(d time.Duration) int {
				return int(d.Nanoseconds() / 1000000)
			},
			"colorize": func(status int) string {
				if status >= 200 && status < 400 {
					return fmt.Sprintf("\x1b[32m%d\x1b[0m", status)
				}
				return fmt.Sprintf("\x1b[31m%d\x1b[0m", status)
			},
		}).
		Parse("\x1b[95mGET\x1b[0m {{.Request.URL}} {{colorize .StatusCode}} {{.ContentType}} {{.ContentLength}} {{ms .Duration}}")
	if err != nil {
		panic(fmt.Sprintf("error parsing response template: %v", err))
	}
	return t
}()

// A Response is represents the outcome of a CrawlRequest.
type Response struct {
	Request       *Request
	Duration      time.Duration
	StatusCode    int
	ContentLength int64
	ContentType   string
	URLs          []string
}

func (c *Response) String() string {
	w := &bytes.Buffer{}
	responseTemplate.Execute(w, c)
	return w.String()
}
