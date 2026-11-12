FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/server


FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server /app/server
EXPOSE 8001

CMD ["/app/server"]