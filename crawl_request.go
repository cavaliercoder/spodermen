package main

import (
	"net/url"
)

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
