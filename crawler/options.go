package crawler

import "context"

type Option func(*Crawler) error

func NoFollow() Option {
	return func(c *Crawler) error {
		c.noFollow = true
		return nil
	}
}

func FollowHosts(hosts ...string) Option {
	return func(c *Crawler) error {
		for _, host := range hosts {
			c.followHosts[host] = true
		}
		return nil
	}
}

func WithContext(ctx context.Context) Option {
	return func(c *Crawler) error {
		c.ctx = ctx
		return nil
	}
}

func UserAgentString(s string) Option {
	return func(c *Crawler) error {
		c.useragent = s
		return nil
	}
}
