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

Repo này đã được tách sẵn thành 2 chart:

- `helm/base` cho app microservice: `order`, `payment`, `stock`
- `helm/monitoring` cho observability: `prometheus`, `grafana`, `alertmanager`

### Cách đóng gói các file trong `manifest/` sang Helm

#### 1. Gom nhóm theo chức năng

- App runtime: `order`, `payment`, `stock`
- Monitoring: `prometheus.yaml`, `prometheus-rbac.yaml`, `grafana.yaml`, `grafana-secret.yaml`, `alertmanager.yaml`, `alertmanager-secret.yaml`

#### 2. Tạo chart và khung thư mục

Nếu chưa có chart, tạo tối thiểu các file sau:

- `Chart.yaml`
- `values.yaml`
- `templates/`
- `templates/_helpers.tpl`

Trong repo này, phần app đã nằm ở `helm/base` và phần monitoring nằm ở `helm/monitoring`.

#### 3. Đưa giá trị cố định vào `values.yaml`

Các giá trị nên đưa ra `values.yaml` gồm:

- image repository và tag
- replica count
- service type
- port
- namespace
- credentials và secret name
- retention, scrape interval, admin user/password

Ví dụ:

```yaml
order:
  image:
    repository: winder510/order
    tag: latest
  port: 8080
```

#### 4. Chuyển YAML tĩnh sang template

Thay các giá trị hard-code bằng template Helm:

- `metadata.namespace` -> `{{ .Release.Namespace }}`
- `spec.replicas` -> `{{ .Values.<component>.replicas }}`
- `image` -> `{{ .Values.<component>.image.repository }}:{{ .Values.<component>.image.tag }}`
- `ports` -> `{{ .Values.<component>.port }}`
- tên service, secret, configmap -> giữ cố định hoặc đưa vào helper nếu cần tái sử dụng

Trong repo này, helper dùng chung nằm ở `helm/base/templates/_helpers.tpl`.

#### 5. Tách manifest theo chart

- `services/order`, `services/payment`, `services/stock` được gom vào `helm/base/templates/apps.yaml`
- `prometheus.yaml` được gom vào `helm/monitoring/templates/prometheus.yaml`
- `grafana.yaml` được gom vào `helm/monitoring/templates/grafana.yaml`
- `alertmanager.yaml` được gom vào `helm/monitoring/templates/alertmanager.yaml`
- `prometheus-rbac.yaml` nên được tách thành một template riêng trong chart monitoring nếu muốn quản lý RBAC bằng Helm
- `grafana-secret.yaml` và `alertmanager-secret.yaml` nên được chuyển thành template `Secret` hoặc lấy từ `values.yaml`/external secret manager

#### 6. Dùng namespace theo release

Không nên hard-code namespace trong manifest. Hãy để Helm quyết định bằng:

```yaml
metadata:
  namespace: {{ .Release.Namespace }}
```

Khi cài đặt, truyền namespace bằng lệnh `helm upgrade --install -n <namespace>`.

#### 7. Kiểm tra bản render trước khi apply

Luôn chạy render để xem manifest cuối cùng:

```bash
helm template microservices-app ./helm/base -n microservices
helm template microservices-monitoring ./helm/monitoring -n microservices
```

Nếu render ổn thì mới cài:

```bash
helm upgrade --install microservices-app ./helm/base -n microservices --create-namespace
helm upgrade --install microservices-monitoring ./helm/monitoring -n monitoring --create-namespace
```

#### 8. Validate chart

Trước khi merge, kiểm tra thêm:

```bash
helm lint ./helm/base
helm lint ./helm/monitoring
```

### Cách cài trên cluster

App:

```bash
helm upgrade --install microservices-app ./helm/base -n microservices --create-namespace
```

Monitoring:

```bash
helm upgrade --install microservices-monitoring ./helm/monitoring -n microservices --create-namespace
```

`helm/base` hiện tạo các service `LoadBalancer` cho `order`, `payment`, `stock`, nên bạn có thể truy cập bên ngoài nếu cluster hỗ trợ.

## Example request

```bash
curl -X POST http://localhost:8080/order \
  -H "Content-Type: application/json" \
  -d '{"order_id":"ord-1001","sku":"sku-1","qty":2,"amount":125.5}'
```
