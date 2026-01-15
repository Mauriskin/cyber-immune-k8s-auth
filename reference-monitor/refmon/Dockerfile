FROM rust:1.85-alpine AS builder

WORKDIR /app

RUN echo "http://dl-cdn.alpinelinux.org/alpine/edge/main" >> /etc/apk/repositories && \
    echo "http://dl-cdn.alpinelinux.org/alpine/edge/community" >> /etc/apk/repositories && \
    apk update && \
    apk add --no-cache musl-dev protoc protobuf-dev

COPY Cargo.toml Cargo.lock* ./
COPY build.rs .
COPY src ./src

RUN cargo build --release --locked

FROM scratch
COPY --from=builder /app/target/release/refmon /refmon
USER 10001:10001
EXPOSE 50051
ENTRYPOINT ["/refmon"]