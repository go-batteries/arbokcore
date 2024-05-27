# cli.Dockerfile
FROM golang:1.22

ENV CGO_ENABLED=1

WORKDIR /app

COPY . .

RUN go mod tidy
RUN go build -ldflags "-w" -o arbokcli cmd/cli/main.go

ENTRYPOINT ["/app/arbokcli"]
