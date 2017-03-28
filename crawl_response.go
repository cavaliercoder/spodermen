package main

import (
	"bytes"
	"text/template"
	"time"
)

const (
	// CrawlResponseTemplate is the template used to print the outcome of a CrawlRequest.
	CrawlResponseTemplate = "GET {{.Request.URL}} {{.StatusCode}} {{.ContentType}} {{.ContentLength}} {{ms .Duration}}"
)

var crawlResponseTemplate = func() *template.Template {
	t, err := template.New("crawl_response").
		Funcs(template.FuncMap{
			"ms": func(d time.Duration) int {
				return int(d.Nanoseconds() / 1000000)
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
	ContentLength int
	ContentType   string
	URLs          []string
}

func (c *CrawlResponse) String() string {
	w := &bytes.Buffer{}
	crawlResponseTemplate.Execute(w, c)
	return w.String()
}
