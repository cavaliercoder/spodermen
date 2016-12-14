package main

import (
	"fmt"
	"time"
)

type CrawlResponse struct {
	Request       *CrawlRequest
	Duration      time.Duration
	StatusCode    int
	ContentLength int
	ContentType   string
}

func (c *CrawlResponse) String() string {
	return fmt.Sprintf("GET %v %v %v %v %v", c.Request.URL.Path, c.StatusCode, c.ContentType, c.ContentLength, int(c.Duration/1000000))
}
