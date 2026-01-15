FROM golang:1.25.5-alpine AS builder
WORKDIR /app

# Установка protobuf и git для gRPC-генерации (если нужно внутри образа)
RUN apk add --no-cache protobuf-dev git

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
COPY refmon/ ./refmon/
COPY security/ ./security/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o token-enforcer .

FROM scratch
COPY --from=builder /app/token-enforcer /token-enforcer
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/token-enforcer"]