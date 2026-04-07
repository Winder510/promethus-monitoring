FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod ./
COPY go.sum ./
COPY services ./services
RUN go mod download
WORKDIR /src/services/order
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/order .

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /out/order /order
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/order"]
