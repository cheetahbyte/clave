FROM golang:alpine AS builder
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -o /app/clave ./cmd/clave

FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/clave /clave
COPY migrations /migrations
EXPOSE 8000
CMD ["/clave"]
