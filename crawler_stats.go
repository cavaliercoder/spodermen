package main

import (
	"encoding/json"
	"sync"
)

// CrawlerStats collects transfer statistics for a Crawler.
type CrawlerStats struct {
	sync.Mutex
	TotalRequests int32
	TotalBytes    int64
	StatusCodes   map[int]int32
	MimeTypes     map[string]int32
}

// AddResponse increments all statistics according to the given response.
func (c *CrawlerStats) AddResponse(resp *CrawlResponse) {
	c.Lock()
	defer c.Unlock()

	c.TotalRequests++
	c.TotalBytes += int64(resp.ContentLength)

	if c.StatusCodes == nil {
		c.StatusCodes = make(map[int]int32, 4)
	}
	c.StatusCodes[resp.StatusCode]++

	if c.MimeTypes == nil {
		c.MimeTypes = make(map[string]int32, 4)
	}
	c.MimeTypes[resp.ContentType]++
}

// JSON returns a JSON representation of the crawler statistics.
func (c *CrawlerStats) JSON() string {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err.Error()
	}

	return string(b)
}
