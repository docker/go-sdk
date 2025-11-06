FROM golang:1.24-alpine3.22@sha256:92b6d90efb23fa6ee3eef6d45f5d06527e56b0d3100e98eb5a1d9b3b8acd9bca as builder
WORKDIR /app
COPY . .
RUN mkdir -p dist
RUN go build -o ./dist/server main.go

FROM alpine
WORKDIR /app
COPY --from=builder /app/tls.pem /app/tls-key.pem ./
COPY --from=builder /app/dist/server .
EXPOSE 6443
CMD ["/app/server"]
