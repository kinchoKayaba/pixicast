FROM golang:1.23-alpine

WORKDIR /app

COPY . .

RUN go build -o server ./cmd/server

CMD ["./server"]



