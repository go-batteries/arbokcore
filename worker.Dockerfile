# worker.Dockerfile
FROM golang:1.22

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -ldflags "-w" -o arbokworker cmd/workers/main.go

ENTRYPOINT ["./arbokworker"]
