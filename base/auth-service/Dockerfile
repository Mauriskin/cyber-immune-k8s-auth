FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY main.go .
COPY security/ ./security/

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o auth-service .

FROM scratch
COPY --from=builder /app/auth-service /auth-service
USER 10001:10001
EXPOSE 8080
ENTRYPOINT ["/auth-service"]