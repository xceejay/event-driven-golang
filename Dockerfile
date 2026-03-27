# Stage 1: Build
FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/engine ./cmd/engine/main.go

# Stage 2: Runtime
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /app/engine /app/engine
COPY migrations/ /app/migrations/
COPY web/ /app/web/
COPY config.railway.yaml /app/config.railway.yaml

EXPOSE 8080

ENTRYPOINT ["/app/engine"]
