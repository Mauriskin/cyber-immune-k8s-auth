# cyber-immune-k8s-auth
Кибериммунная система аутентификации и управления доступом на Kubernetes
Проект представляет собой прототип кибериммунной системы аутентификации, спроектированной по принципам secure by design.
Основная идея — даже при компрометации отдельных компонентов критические активы (секрет подписи JWT, учётные данные) остаются защищёнными, а система сохраняет работоспособность ключевых функций.

## Описание проекта

Проект представляет собой прототип кибериммунной системы аутентификации, спроектированной по принципу **secure by design**.  
Даже при компрометации низкодоверенных компонентов (Ingress или Auth Service) критические активы (секрет подписи JWT) остаются защищёнными, а система сохраняет работоспособность.

Ключевые особенности:
- **Минимальная TCB** — Reference Monitor на Rust (<250 строк, scratch-образ ~6 MB).
- **Reference Monitor** — единственный компонент с доступом к секрету JWT (gRPC, без HTTP/JSON).
- **Security Controller** — OPA с Rego-политиками (default deny + explicit allow).
- **Трёхуровневая изоляция** (Untrusted → Medium → High/TCB) с Cilium, gVisor и restricted PSA.
- **Защита от угроз**: lateral movement, container escape, tampering, DoS.
- **Мониторинг**: Falco + Prometheus + Loki + Grafana.

## Quick Start (Быстрая установка)

Проект протестирован на Ubuntu 22.04/24.04 с Minikube (kvm2 driver).

### Требования
- CPU: 6+ ядер, RAM: 16 GB
- Docker, Minikube, kubectl, Helm 3

### Автоматизированная установка (рекомендуется)

В репозитории есть скрипт `start-config.sh` для полной установки и развёртывания.

```bash
# 1. Клонирование репозитория
git clone https://github.com/your-username/cyber-immune-auth.git
cd cyber-immune-auth

# 2. Запуск скрипта
chmod +x start-config.sh
./start-config.sh

Скрипт выполнит:

Сборку образов (Reference Monitor, Token Enforcer, Auth Service).
Загрузку образов в Minikube.
Установку NGINX Ingress и OPA Gatekeeper.
Создание namespaces с restricted PSA.
Применение Gatekeeper-политики gVisor.
Развёртывание всех компонентов, NetworkPolicy и Ingress.
Ожидание готовности подов.

Ручная установка (если нужно)

# Запуск Minikube
minikube start --driver=kvm2 --cpus=6 --memory=16384mb --container-runtime=containerd
minikube addons enable gvisor

# Сборка и загрузка образов
eval $(minikube docker-env)
docker build -t secure-refmon:latest ./reference-monitor/refmon
docker build -t token-enforcer:latest ./base/token_enforcer
docker build -t secure-auth-service:latest ./base/auth-service
minikube image load secure-refmon:latest token-enforcer:latest secure-auth-service:latest

# Установка компонентов
helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx
helm repo add gatekeeper https://open-policy-agent.github.io/gatekeeper/charts
helm repo update

helm upgrade --install nginx-ingress ingress-nginx/ingress-nginx --namespace ingress-nginx --create-namespace
helm upgrade --install gatekeeper gatekeeper/gatekeeper --namespace gatekeeper-system --create-namespace

# Развёртывание
kubectl apply -f manifests/

Доступ к API

minikube tunnel  # в отдельном терминале
INGRESS_IP=$(minikube ip)
INGRESS_PORT=$(kubectl -n ingress-nginx get svc nginx-ingress-controller -o jsonpath='{.spec.ports[?(@.name=="http")].nodePort}')
echo "API URL: http://$INGRESS_IP:$INGRESS_PORT/login"

curl -X POST http://$INGRESS_IP:$INGRESS_PORT/login \
  -d '{"username":"admin","password":"secret123","mfa":"123456"}' \
  -H "Content-Type: application/json"

cyber-immune-auth/
├── base/                      # Auth Service и Token Enforcer
├── reference-monitor/         # Rust TCB (Reference Monitor)
├── manifests/                 # Kubernetes манифесты
├── policy/                    # Rego-политики и YAML
├── tests/                     # k6 тесты
├── start-config.sh            # Автоматизация установки
└── README.md

Лицензия
MIT License
