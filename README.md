# Test Repository

## Useful links
Get started with otel [link](https://opentelemetry.io/docs/instrumentation/go/getting-started/)

Jaeger Operator 1.30 [link](https://www.jaegertracing.io/docs/1.30/operator/)

## GO
- `docker-compose up -d`

## Access
- `127.0.0.1:8080/TCP` Jaeger UI
- `127.0.0.1:6831/UDP` Jaeger Agent

## Screenshots
![screenshot_1](assets/screen1.png)
![screenshot_2](assets/screen2.png)

## Kind + Operator

```bash
kind create cluster --config kind-1.22.yaml --name observability-test
kubectl create namespace observability
kubectl create -f https://github.com/jaegertracing/jaeger-operator/releases/download/v1.30.0/jaeger-operator.yaml -n observability
kubectl apply -f simplest.yaml
docker build -t acme/myapp:myversion .
kind load docker-image acme/myapp:myversion --name=observability-test
kubectl apply -f deploy.yaml
k port-forward service/simplest-query 8080:16686
```
