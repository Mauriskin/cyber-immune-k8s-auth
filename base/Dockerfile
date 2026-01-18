# syntax=docker/dockerfile:1
FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /app/service .

FROM scratch

COPY --from=builder /app/service /service

USER 10001:10001

ENTRYPOINT ["/service"]

EXPOSE 8080
