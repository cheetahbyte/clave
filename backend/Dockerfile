FROM golang:alpine AS builder
WORKDIR /build
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /app/clave ./cmd/clave

FROM scratch
COPY --from=builder /app/clave /clave
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /build/migrations /migrations
EXPOSE 8000
USER 65532:65532
CMD ["/clave"]
