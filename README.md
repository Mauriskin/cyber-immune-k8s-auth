# cyber-immune-k8s-auth
Кибериммунная система аутентификации и управления доступом на Kubernetes

## Описание проекта

Проект представляет собой прототип кибериммунной системы аутентификации, спроектированной по принципу **secure by design**.  
Даже при компрометации низкодоверенных компонентов (Ingress или Auth Service) критические активы (секрет подписи JWT) остаются защищёнными, а система сохраняет работоспособность.

Ключевые особенности:
- **Минимальная TCB** — Reference Monitor на Rust (<250 строк, scratch-образ ~6 MB).
- **Reference Monitor** — единственный компонент с доступом к секрету JWT (gRPC, без HTTP/JSON).
- **Security Controller** — OPA с Rego-политиками (default deny + explicit allow).
- **Трёхуровневая изоляция** (Untrusted → Medium → High) с Cilium, gVisor и restricted PSA.
- **Защита от угроз**: lateral movement, container escape, tampering, DoS.
- **Мониторинг**: Falco + Prometheus + Loki + Grafana.


## Архитектура

Внешний клиент
↓ (HTTPS)
NGINX Ingress (Домен 1 — Untrusted)
↓
Auth Service (Домен 2 — Medium): проверка credentials + rate limiting
↓ (HTTP/JSON)
Token Enforcer (Домен 3 — High): прокси + OPA (Security Controller)
↓ (gRPC + Protobuf)
Reference Monitor (Rust TCB): только криптография JWT (HS256, TTL=60s)


## Быстрая установка

Проект протестирован на Ubuntu 22.04/24.04 с Minikube на kvm2.

### Требования
- CPU: 6+ ядер, RAM: 16 GB (желательно)
- Docker, Minikube, kubectl, Helm 3

### Автоматизированная установка

В репозитории есть скрипт `start-config.sh` для полной установки и развёртывания.

```bash
# 1. Клонирование репозитория
git clone https://github.com/your-username/cyber-immune-auth.git
cd cyber-immune-auth

# 2. Запуск скрипта
chmod +x start-config.sh
./start-config.sh
```
Скрипт выполнит:

Сборку образов (Reference Monitor, Token Enforcer, Auth Service).
Загрузку образов в Minikube.
Установку NGINX Ingress и OPA Gatekeeper.
Создание namespaces с restricted PSA.
Применение Gatekeeper-политики gVisor.
Развёртывание всех компонентов, NetworkPolicy и Ingress.
Ожидание готовности подов.

Доступ к API
```bash
minikube tunnel  # в отдельном терминале
INGRESS_IP=$(minikube ip)
INGRESS_PORT=$(kubectl -n ingress-nginx get svc nginx-ingress-controller -o jsonpath='{.spec.ports[?(@.name=="http")].nodePort}')
echo "API URL: http://$INGRESS_IP:$INGRESS_PORT/login"
```
Тест
```bash
curl -X POST http://$INGRESS_IP:$INGRESS_PORT/login \
  -d '{"username":"admin","password":"secret123","mfa":"123456"}' \
  -H "Content-Type: application/json"
```
Автоматизированные тесты k6 в директории tests/:
```bash
./run-tests.sh
```
Тесты покрывают:

Позитивные сценарии (успешная аутентификация).
Негативные (неверные данные, прямой доступ к TCB).
Пограничные (rate limiting, истечение токена).

cyber-immune-auth/
├── base/                      # Auth Service и Token Enforcer
├── reference-monitor/         # Rust TCB (Reference Monitor)
├── manifests/                 # Kubernetes манифесты
├── policy/                    # Rego-политики и YAML
├── tests/                     # k6 тесты
├── start-config.sh            # Автоматизация установки
└── README.md

Лицензия
MIT License — свободное использование и модификация.
