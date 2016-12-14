package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	USER_AGENT = "Spodermen bot/1.0"
)

var (
	// regexp pattern to match href anchors
	pattern = regexp.MustCompile(`href\s*=\s*"([^"]*)"`)
)

type Crawler interface {
	Do(*CrawlRequest) (*CrawlResponse, error)
	Next() (*CrawlResponse, error)
	Start(...*CrawlRequest)
}

type crawler struct {
	client *http.Client
	queue  Queue
}

func NewCrawler() Crawler {
	return &crawler{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 20,
			},
			Timeout: time.Duration(time.Second * 10),
		},
		queue: NewQueue(),
	}
}

func (c *crawler) Start(reqs ...*CrawlRequest) {
	for _, req := range reqs {
		c.queue.Enqueue(req)
	}
}

func (c *crawler) Next() (*CrawlResponse, error) {
	req := c.queue.Dequeue()
	return c.Do(req)
}

func (c *crawler) Do(req *CrawlRequest) (*CrawlResponse, error) {
	resp := &CrawlResponse{
		Request: req,
	}

	hreq, err := http.NewRequest("GET", req.URL.String(), nil)
	if err != nil {
		return nil, err
	}
	hreq.Header.Set("User-Agent", USER_AGENT)

	start := time.Now()
	hresp, err := c.client.Do(hreq)
	if err != nil {
		return nil, err
	}

	body := ""
	if err := func() error {
		defer hresp.Body.Close()
		b, err := ioutil.ReadAll(hresp.Body)
		body = string(b)
		resp.ContentLength = len(b)
		return err
	}(); err != nil {
		return nil, err
	}

	resp.Duration = time.Since(start)
	resp.StatusCode = hresp.StatusCode
	resp.ContentType = hresp.Header.Get("Content-Type")
	if resp.ContentType == "" {
		resp.ContentType = "-"
	} else {
		// strip after ;
		if n := strings.Index(resp.ContentType, ";"); n != -1 {
			resp.ContentType = resp.ContentType[:n]
		}
	}

	hrefs, err := getHrefs(strings.NewReader(body))
	if err != nil {
		return nil, err
	}

	for _, href := range hrefs {
		// TODO: match all domain-local URIs
		if strings.HasPrefix(href, "/") {
			// TODO: deep copy target.URL.User
			uri := *req.URL
			uri.Path = href

			c.queue.Enqueue(&CrawlRequest{
				URL: &uri,
			})
		}
	}

	return resp, nil
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
