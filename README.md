# cyber-immune-k8s-auth
Кибериммунная система аутентификации и управления доступом на Kubernetes

## Описание проекта

Проект представляет собой прототип кибериммунной системы аутентификации, спроектированной по принципу **secure by design**.  
Даже при компрометации низкодоверенных компонентов (Ingress или Auth Service) критические активы (секрет подписи JWT) остаются защищёнными, а система сохраняет работоспособность.

Ключевые особенности:
- **Минимальная TCB**: Reference Monitor на Rust — единственный компонент с доступом к секрету подписи JWT (scratch-образ, без shell).
- **Строгая изоляция**: 3 домена (Untrusted → Medium → High) с Cilium NetworkPolicy (default deny) и gVisor sandbox.
- **Референсный монитор политик**: OPA с Rego (2 файла политик) — проверяет каждое взаимодействие перед доступом к TCB.
- **Защита от угроз**:
  - Lateral movement: блокируется NetworkPolicy и OPA.
  - Container escape/tampering: gVisor + restricted PSA + non-root.
  - DoS: multi-level rate limiting (Ingress + приложение).
  - Подделка JWT: короткий TTL (60с) + секрет только в TCB.
- **Непрерывная верификация**: Falco (runtime detection) + Prometheus/Loki/Grafana (логи и метрики).


## Архитектура

<img width="1113" height="391" alt="image" src="https://github.com/user-attachments/assets/7d1bf6ca-ba42-477b-bee4-896749ec26f8" />



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

1. Сборку образов (Reference Monitor, Token Enforcer, Auth Service).
2. Загрузку образов в Minikube.
3. Установку NGINX Ingress и OPA Gatekeeper.
4. Создание namespaces с restricted PSA.
5. Применение Gatekeeper-политики gVisor.
6. Развёртывание всех компонентов, NetworkPolicy и Ingress.
7. Ожидание готовности подов.

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

1. Позитивные сценарии (успешная аутентификация).
2. Негативные (неверные данные, прямой доступ к TCB).
3. Пограничные (rate limiting, истечение токена).

Структура репозитория

- base/ — Auth Service и Token Enforcer (Go-код и Dockerfiles).
- reference-monitor/ — Rust Reference Monitor (TCB).
- security-controller/ — OPA + Rego-политики.
- manifests/ — Kubernetes манифесты (deployments, NetworkPolicy, Ingress).
- policy/ — YAML-политики и генераторы.
- tests/ — k6 тесты.
- start-config.sh — Автоматизация установки.

Лицензия
MIT License — свободное использование и модификация.
