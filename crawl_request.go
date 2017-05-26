package main

import (
	"net/url"
)

// A CrawlRequest represents a single URL in a crawling queue.
type CrawlRequest struct {
	URL    *url.URL
	Follow bool
}

// A CrawlResponseCallback is a function which may be called by the Crawler
// once a page has been crawled.
type CrawlResponseCallback func(*CrawlResponse) error

func NewCrawlRequest(urlStr string, follow bool) (*CrawlRequest, error) {
	uri, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	return &CrawlRequest{uri, follow}, nil
}

func (c *CrawlRequest) String() string {
	return c.URL.String()
}
