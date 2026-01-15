#!/bin/bash
set -e  # Остановка при ошибке

echo "=== Восстановление кибериммунной системы ==="

# Путь к проекту (измените, если другой)
PROJECT_DIR=~/cyber-immune-auth

# 1. Helm репозитории
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo add gatekeeper https://open-policy-agent.github.io/gatekeeper/charts
helm repo update

# 2. Сборка образов локально (без eval docker-env)
echo "Сборка Reference Monitor (TCB)..."
cd $PROJECT_DIR/reference-monitor/refmon
docker build -t secure-refmon:latest .

echo "Сборка Token Enforcer..."
cd $PROJECT_DIR/base/token_enforcer
docker build -t token-enforcer:latest .

echo "Сборка Auth Service..."
cd $PROJECT_DIR/base/auth-service
docker build -t secure-auth-service:latest .

# 3. Загрузка образов в Minikube (minikube image load)
echo "Загрузка образов в Minikube..."
minikube image load secure-refmon:latest
minikube image load token-enforcer:latest
minikube image load secure-auth-service:latest

# 4. Установка NGINX Ingress Controller
echo "Установка NGINX Ingress..."
helm upgrade --install nginx-ingress ingress-nginx/ingress-nginx \
  --namespace ingress-nginx --create-namespace \
  --set controller.allowSnippetAnnotations=true \
  --set controller.runtimeClassName=gvisor \
  --set controller.service.type=NodePort

# 5. Установка OPA Gatekeeper
echo "Установка Gatekeeper..."
helm upgrade --install gatekeeper gatekeeper/gatekeeper \
  --namespace gatekeeper-system --create-namespace

# 6. Создание namespaces с restricted PSA
echo "Создание namespaces..."
kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: domain1-untrusted
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
    security-domain: untrusted
---
apiVersion: v1
kind: Namespace
metadata:
  name: domain2-medium
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
    security-domain: medium
---
apiVersion: v1
kind: Namespace
metadata:
  name: domain3-tcb
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
    security-domain: tcb
EOF

# 7. Принудительный gVisor для всех подов в доменах (Gatekeeper policy)
echo "Применение Gatekeeper политики gVisor..."
kubectl apply -f - <<EOF
apiVersion: templates.gatekeeper.sh/v1
kind: ConstraintTemplate
metadata:
  name: requiregvisor
spec:
  crd:
    spec:
      names:
        kind: RequireGVisor
  targets:
    - target: admission.k8s.gatekeeper.sh
      rego: |
        package requiregvisor
        violation[{"msg": msg}] {
          not input.review.object.spec.runtimeClassName
          msg := "All pods must use gvisor runtimeClass"
        }
        violation[{"msg": msg}] {
          input.review.object.spec.runtimeClassName != "gvisor"
          msg := sprintf("Forbidden runtimeClass: %v, must be gvisor", [input.review.object.spec.runtimeClassName])
        }
---
apiVersion: constraints.gatekeeper.sh/v1beta1
kind: RequireGVisor
metadata:
  name: require-gvisor-domains
spec:
  match:
    kinds:
      - apiGroups: [""]
        kinds: ["Pod"]
    namespaces:
      - "domain1-untrusted"
      - "domain2-medium"
      - "domain3-tcb"
EOF

# 8. Развёртывание компонентов
echo "Развёртывание Reference Monitor..."
kubectl apply -f $PROJECT_DIR/reference-monitor/refmon/refmon-deployment.yaml

echo "Развёртывание Token Enforcer..."
kubectl apply -f $PROJECT_DIR/base/token_enforcer/token-enforcer-deployment.yaml

echo "Развёртывание Auth Service..."
kubectl apply -f $PROJECT_DIR/base/auth-service/auth-deployment.yaml

# 9. NetworkPolicy и Ingress
echo "Применение NetworkPolicy..."
kubectl apply -f $PROJECT_DIR/network-policies.yaml  # предположение, измените если другой путь

kubectl apply -f $PROJECT_DIR/reference-monitor/refmon/networkpolicy-refmon.yaml

echo "Применение Ingress..."
kubectl apply -f $PROJECT_DIR/base/auth-service/auth-ingress.yaml

# 10. Ожидание готовности
kubectl wait --for=condition=Ready pod -l app=reference-monitor -n domain3-tcb --timeout=300s
kubectl wait --for=condition=Ready pod -l app=token-enforcer -n domain3-tcb --timeout=300s
kubectl wait --for=condition=Ready pod -l app=auth-service -n domain2-medium
