FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /pinwheel ./cmd/proxy/main.go


FROM alpine:3.20

RUN adduser -D -g '' appuser

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY --from=builder /pinwheel /app/pinwheel

RUN chown -R appuser:appuser /app

USER appuser

WORKDIR /app

EXPOSE 8080

ENTRYPOINT ["./pinwheel"]
