FROM golang:1.22-alpine AS builder
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod ./
COPY go.sum ./
COPY services ./services
RUN go mod download
WORKDIR /src/services/stock
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/stock .

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /out/stock /stock
EXPOSE 8082
USER nonroot:nonroot
ENTRYPOINT ["/stock"]
