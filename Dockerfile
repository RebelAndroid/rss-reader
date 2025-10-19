FROM golang:latest AS builder
WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY static ./static
COPY templates ./templates
COPY migrations ./migrations

COPY api.go .
COPY db.go .
COPY events.go .
COPY main.go .

RUN go build

CMD ["./rss-reader"]
