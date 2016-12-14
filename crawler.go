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

func NewCrawlRequest(urlStr string) (*CrawlRequest, error) {
	uri, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	return &CrawlRequest{uri}, nil
}

func crawl(target *CrawlRequest) <-chan *CrawlRequest {
	ch := make(chan *CrawlRequest, 0)
	go func() {
		defer close(ch)

		req, err := http.NewRequest("GET", target.URL.String(), nil)
		panicOn(err) // TODO: smarter handling of bad URLs

		start := time.Now()
		resp, err := http.DefaultClient.Do(req)
		panicOn(err) // TODO: smarter handling of transit errors

		body := ""
		length := -1
		if err := func() error {
			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			body = string(b)
			length = len(b)
			return err
		}(); err != nil {
			errorf("%v\n", err)
			return
		}

		ms := int(time.Since(start) / 1000000)
		printf("%v %v %v %v %v\n", req.Method, req.URL.Path, resp.StatusCode, length, ms)
		hrefs, err := getHrefs(strings.NewReader(body))
		panicOn(err) // TODO: smarted handling of parser errors

		for _, href := range hrefs {
			// TODO: match all domain-local URIs
			if strings.HasPrefix(href, "/") {
				// TODO: deep copy target.URL.User
				uri := *target.URL
				uri.Path = href

				ch <- &CrawlRequest{
					URL: &uri,
				}
			}
		}
	}()

	return ch
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
