FROM golang:alpine AS builder
WORKDIR /build
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN mkdir -p /app && \
    CGO_ENABLED=0 go build -o /app/clave ./cmd/clave && \
    CGO_ENABLED=0 go build -o /app/createadmin ./cmd/createadmin && \
    CGO_ENABLED=0 go build -o /app/resetadmin2fa ./cmd/resetadmin2fa && \
    CGO_ENABLED=0 go build -o /app/deltaworker ./cmd/deltaworker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/clave /clave
COPY --from=builder /app/createadmin /createadmin
COPY --from=builder /app/resetadmin2fa /resetadmin2fa
COPY --from=builder /build/migrations /migrations
COPY --from=builder /app/deltaworker /deltaworker
EXPOSE 8000
CMD ["/clave"]
