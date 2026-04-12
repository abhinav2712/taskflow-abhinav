# Stage 1: build the binary
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Cache deps separately from source — only re-downloaded when go.mod/go.sum change
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /taskflow-api ./cmd/server

# Stage 2: minimal runtime image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /taskflow-api /taskflow-api

EXPOSE 8080

ENTRYPOINT ["/taskflow-api"]
