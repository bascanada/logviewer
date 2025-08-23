# Multi-stage build
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o logviewer .

FROM gcr.io/distroless/static-debian11
COPY --from=builder /app/logviewer /logviewer
EXPOSE 8080
ENTRYPOINT ["/logviewer", "server"]
CMD ["--config", "/config/config.json", "--port", "8080", "--host", "0.0.0.0", "--logging-stdout"]
