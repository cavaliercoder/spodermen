package main

import (
	"bytes"
	"text/template"
	"time"
)

const (
	CRAWL_RESPONSE_TEMPLATE = "GET {{.Request.URL.Path}} {{.StatusCode}} {{.ContentType}} {{.ContentLength}} {{ms .Duration}}"
)

var (
	crawlResponseTemplate = template.New("crawl_response")
)

func initOutput() error {
	t, err := template.New("crawl_response").
		Funcs(template.FuncMap{
			"ms": func(d time.Duration) int {
				return int(d.Nanoseconds() / 1000000)
			},
		}).
		Parse(CRAWL_RESPONSE_TEMPLATE)
	if err != nil {
		return err
	}

	crawlResponseTemplate = t

	return nil
}

type CrawlResponse struct {
	Request       *CrawlRequest
	Duration      time.Duration
	StatusCode    int
	ContentLength int
	ContentType   string
}

func (c *CrawlResponse) String() string {
	w := &bytes.Buffer{}
	crawlResponseTemplate.Execute(w, c)
	return w.String()
}
