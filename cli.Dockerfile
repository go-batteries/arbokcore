# cli.Dockerfile
FROM golang:1.22-alpine

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -ldflags "-w" -o arbokcli cmd/cli/main.go
