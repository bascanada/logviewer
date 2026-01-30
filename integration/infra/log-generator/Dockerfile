# Use the same Go version as your project for consistency
FROM golang:1.24.3-alpine AS builder

WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .

RUN go build -o /log-generator .

FROM alpine:latest
WORKDIR /
COPY --from=builder /log-generator /log-generator
ENTRYPOINT ["/log-generator"]
