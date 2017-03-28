package main

import (
	"net/url"
)

// A CrawlRequest represents a single URL in a crawling queue.
type CrawlRequest struct {
	URL      *url.URL
	Callback CrawlResponseCallback
}

// A CrawlResponseCallback is a function which may be called by the Crawler
// once a page has been crawled.
type CrawlResponseCallback func(*CrawlResponse) error

func NewCrawlRequest(urlStr string, callback CrawlResponseCallback) (*CrawlRequest, error) {
	uri, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	return &CrawlRequest{uri, callback}, nil
}

func (c *CrawlRequest) String() string {
	return c.URL.String()
}
