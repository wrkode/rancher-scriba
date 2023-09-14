FROM golang:1.21.1-bookworm

# RUN apk add --update --no-cache ca-certificates

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o scriba .

CMD ["./scriba"]