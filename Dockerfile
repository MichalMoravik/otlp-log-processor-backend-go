FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o otlp-log-processor

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/otlp-log-processor .
EXPOSE 4317
CMD ["./otlp-log-processor"] 