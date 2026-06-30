FROM golang:1.23-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /build/server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/server .
EXPOSE 9090
ENTRYPOINT ["./server"]
