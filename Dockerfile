FROM golang:1.26-alpine AS builder

WORKDIR /build

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o app ./cmd

FROM alpine:3.22
COPY --from=builder /build/app /app
ENV DSN=/data/app.db
VOLUME /data
ENTRYPOINT ["/app"]
