# Microservice Base

Boilerplate Go microservices with basic service-to-service calls and separate Helm releases for the app and observability stack.

## Services

- `order` receives an order request and calls `payment`
- `payment` authorizes payment and calls `stock`
- `stock` reserves inventory

## Local run

Each service reads its port from `PORT`.

```bash
go run ./services/stock
```

The default ports are:

- `order`: `8080`
- `payment`: `8081`
- `stock`: `8082`

## Docker

Build each service image from its own folder:

```bash
docker build -t order:latest -f services/order/Dockerfile .
docker build -t payment:latest -f services/payment/Dockerfile .
docker build -t stock:latest -f services/stock/Dockerfile .
```

## Helm

The app chart lives in `helm/base` and installs the Go services:

- order
- payment
- stock

Each service is exposed as a Kubernetes `LoadBalancer` Service, so you can reach them externally if your cluster supports it.

Install the app with:

```bash
helm upgrade --install microservices-app ./helm/base -n microservices --create-namespace
```

The chart does not create the namespace itself when you use `--create-namespace`.

The monitoring chart lives in `helm/monitoring` and installs:

- Prometheus
- OpenTelemetry Collector
- ClickHouse
- Grafana
- Alertmanager

Install it separately:

```bash
helm upgrade --install microservices-monitoring ./helm/monitoring -n microservices
```

## Example request

```bash
curl -X POST http://localhost:8080/order \
  -H "Content-Type: application/json" \
  -d '{"order_id":"ord-1001","sku":"sku-1","qty":2,"amount":125.5}'
```
