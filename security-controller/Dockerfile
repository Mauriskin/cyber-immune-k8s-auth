FROM golang:1.25.5-alpine AS builder
WORKDIR /app

# Установка protoc и плагинов
RUN apk add --no-cache protobuf protobuf-dev

# Установка protoc-gen-go и protoc-gen-go-grpc через go install
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Добавляем $GOPATH/bin в PATH
ENV PATH="$PATH:/go/bin"

COPY go.mod go.sum ./
RUN go mod download

COPY *.go *.proto ./

# Генерация кода из proto
RUN protoc --go_out=. --go-grpc_out=. security.proto

# Сборка бинарника
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o security-controller .

FROM scratch
COPY --from=builder /app/security-controller /security-controller
USER 10001:10001
EXPOSE 50052
ENTRYPOINT ["/security-controller"]