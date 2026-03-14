FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o soms-client main.go

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/soms-client .
COPY --from=builder /app/web ./web

# Ensure keys directory exists
RUN mkdir -p keys

ENV SOMS_SERVER_URL=http://server:5000
ENV SOMS_CLIENT_ADDR=:8080

EXPOSE 8080

CMD ["./soms-client"]
