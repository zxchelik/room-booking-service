FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /service ./cmd/service

FROM alpine:3.21

RUN adduser -D -u 10001 appuser
WORKDIR /app
USER appuser

COPY --from=builder /service /service
COPY api.yaml /app/api.yaml

EXPOSE 8080

ENTRYPOINT ["/service"]
