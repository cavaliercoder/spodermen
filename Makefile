prefix := /usr/local
exec_prefix := $(prefix)
bindir := $(exec_prefix)/bin

all: spodermen

SOURCES = \
	main.go \
	crawl_request.go \
	crawl_response.go \
	crawler.go \
	crawler_stats.go

spodermen: $(SOURCES)
	go build -x -o spodermen $(SOURCES)

clean:
	go clean -x

install:
	install -v -C -m 0755 spodermen $(DESTDIR)$(bindir)/spodermen

.PHONY: all clean
