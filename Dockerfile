FROM golang:alpine3.24 as builder

WORKDIR /build

RUN apk add --no-cache git
RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o placehoder ./cmd/server

FROM scratch
COPY --from=builder /build/bureau /bureau
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

EXPOSE 8081 # placeholder port

ENTRYPOINT ["./placeholder"]

