FROM golang:1.27.3-alpine3.24@sha256:b8c9b82affbbb82a020bbcf998dad2ba9516944ac7e883524642a481a9505e9c as builder
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
