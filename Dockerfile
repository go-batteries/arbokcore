# Dockerfile
FROM golang:1.22-alpine

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -ldflags "-w" -o arbokcore cmd/server/main.go

EXPOSE 9191

CMD ["./arbokcore"]

