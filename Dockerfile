FROM docker.io/library/golang:1.22-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . .

RUN go build -o main cmd/server/main.go

EXPOSE 8080

CMD ["./main"] 