package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// regexp pattern to match href anchors
var pattern = regexp.MustCompile(`href\s*=\s*"([^"]*)"`)

type CrawlRequest struct {
	URL *url.URL
}

type CrawlResponse struct {
	Request       *CrawlRequest
	Duration      time.Duration
	StatusCode    int
	ContentLength int
}

func (c *CrawlResponse) String() string {
	return fmt.Sprintf("GET %v %v %v %v", c.Request.URL.Path, c.StatusCode, c.ContentLength, int(c.Duration/1000000))
}

func NewCrawlRequest(urlStr string) (*CrawlRequest, error) {
	uri, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	return &CrawlRequest{uri}, nil
}

func crawl(queue Queue) (*CrawlResponse, error) {
	target := queue.Dequeue()
	cresp := &CrawlResponse{
		Request: target,
	}

	req, err := http.NewRequest("GET", target.URL.String(), nil)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	body := ""
	if err := func() error {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		body = string(b)
		cresp.ContentLength = len(b)
		return err
	}(); err != nil {
		return nil, err
	}

	cresp.Duration = time.Since(start)
	cresp.StatusCode = resp.StatusCode

	hrefs, err := getHrefs(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	for _, href := range hrefs {
		// TODO: match all domain-local URIs
		if strings.HasPrefix(href, "/") {
			// TODO: deep copy target.URL.User
			uri := *target.URL
			uri.Path = href

			queue.Enqueue(&CrawlRequest{
				URL: &uri,
			})
		}
	}

	return cresp, nil
}

func getHrefs(r io.Reader) ([]string, error) {
	if r == nil {
		return nil, fmt.Errorf("Reader is nil")
	}

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	matches := pattern.FindAllSubmatch(b, -1)
	if matches != nil {
		reqs := make([]string, len(matches))
		for i, match := range matches {
			reqs[i] = string(match[1])
		}

		return reqs, nil
	}

	return nil, nil
}
