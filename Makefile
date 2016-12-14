all: spodermen

spodermen_files = \
	main.go \
	crawl_request.go \
	crawl_response.go \
	crawler.go \
	queue.go

spodermen: $(spodermen_files)
	go build -x -o spodermen $(spodermen_files)

clean:
	go clean -x

.PHONY: all clean
