package main

import (
	"bytes"
	"fmt"
	"text/template"
	"time"
)

const (
	// CrawlResponseTemplate is the template used to print the outcome of a CrawlRequest.
	CrawlResponseTemplate = "\x1b[95mGET\x1b[0m {{.Request.URL}} {{colorize .StatusCode}} {{.ContentType}} {{.ContentLength}} {{ms .Duration}}"
)

var crawlResponseTemplate = func() *template.Template {
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
		Parse(CrawlResponseTemplate)
	if err != nil {
		panic(err)
	}

	return t
}()

// A CrawlResponse is represents the outcome of a CrawlRequest.
type CrawlResponse struct {
	Request       *CrawlRequest
	Duration      time.Duration
	StatusCode    int
	ContentLength int64
	ContentType   string
	URLs          []string
}

func (c *CrawlResponse) String() string {
	w := &bytes.Buffer{}
	crawlResponseTemplate.Execute(w, c)
	return w.String()
}
