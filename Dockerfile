FROM golang:1.10
RUN go get github.com/cavaliercoder/spodermen
WORKDIR /go/src/github.com/cavaliercoder/spodermen
RUN go build -o spodermen main.go
ENV PATH="$PATH:."
CMD ["spodermen"]
