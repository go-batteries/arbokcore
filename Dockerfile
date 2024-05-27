# Dockerfile
FROM golang:1.22

WORKDIR /app

COPY . .

RUN echo "does this work"

RUN go mod tidy
RUN go build -ldflags "-w" -o arbokcore cmd/server/main.go

RUN echo $(ls)

EXPOSE 9191

CMD ["./arbokcore"]

