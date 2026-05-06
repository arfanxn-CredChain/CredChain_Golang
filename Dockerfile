# ---- build stage ----
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git build-base
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ENV CGO_ENABLED=0 GOOS=linux GOARCH=amd64
RUN go build -ldflags="-s -w" -o /server ./main.go

# ---- final stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata && \
    adduser -D -h /app app

WORKDIR /app
COPY --from=builder /server .
COPY --from=builder /src/infrastructure/database/migrations ./infrastructure/database/migrations
COPY --from=builder /src/locales ./locales

RUN mkdir -p /app/logs && \
    chown -R app:app /app

USER app

EXPOSE 8080

CMD ["./server", "serve"]