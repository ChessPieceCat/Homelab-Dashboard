# Build stage
FROM docker.io/library/golang:1.26.5 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o dashboard .

# Runtime stage
FROM docker.io/library/alpine:latest

WORKDIR /app

COPY --from=builder /app/dashboard .
COPY --from=builder /app/web ./web

EXPOSE 8080

CMD ["./dashboard"]