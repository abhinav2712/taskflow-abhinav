FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /taskflow-api ./cmd/server

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /taskflow-api /taskflow-api

EXPOSE 8080

ENTRYPOINT ["/taskflow-api"]
