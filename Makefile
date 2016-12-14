all: spodermen

spodermen_files = \
	main.go \
	crawler.go \
	queue.go

spodermen: $(spodermen_files)
	go build -x -o spodermen $(spodermen_files)

clean:
	go clean -x

.PHONY: all clean
