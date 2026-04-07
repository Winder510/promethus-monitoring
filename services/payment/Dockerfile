FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod ./
COPY go.sum ./
COPY services ./services
RUN go mod download
WORKDIR /src/services/payment
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/payment .

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /out/payment /payment
EXPOSE 8081
USER nonroot:nonroot
ENTRYPOINT ["/payment"]
