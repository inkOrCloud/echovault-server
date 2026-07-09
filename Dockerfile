# Stage 1: Build dependencies (cached layer)
FROM golang:1.23-alpine AS deps
WORKDIR /build
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Stage 2: Build
FROM deps AS builder
WORKDIR /build
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go build -ldflags="-s -w" -o /build/server ./cmd/server

# Stage 3: Runtime
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /build/server .
EXPOSE 9090
ENTRYPOINT ["./server"]
