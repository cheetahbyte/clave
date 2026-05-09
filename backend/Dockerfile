FROM golang:alpine AS builder
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
RUN CGO_ENABLED=0 go build -o /app/clave ./cmd/clave

FROM scratch
COPY --from=builder /app/clave /clave
COPY migrations /migrations
EXPOSE 8000
USER 65532:65532
CMD ["/clave"]
