# Wishlist-service в minikube

Команды выполняются из корня `wishlist-service`. Профиль minikube — `sre`, namespace приложения — `wishlist`.

## Файлы

| Файл | Назначение |
|---|---|
| `namespace.yaml` | Namespace приложения |
| `configmap.yaml` | Несекретные настройки |
| `secret.example.yaml` | Шаблон Secret для локального заполнения |
| `postgres.yaml` | StatefulSet PostgreSQL, headless Service и шаблон PVC |
| `migrate.yaml` | Job миграций |
| `app.yaml` | Deployment приложения и Service NodePort |

## 1. Кластер и образ

```bash
minikube start -p sre --driver=docker --container-runtime=docker
kubectl --context=sre get nodes

eval "$(minikube -p sre docker-env -u)"
docker build -t filimonq/wishlist-service:k8s-v1 .
```

После успешной сборки загрузите образ в minikube и проверьте его наличие:

```bash
minikube -p sre image load filimonq/wishlist-service:k8s-v1
minikube -p sre image ls | grep wishlist-service
```

Образ содержит backend, встроенный frontend и команду миграций. Здесь собираем его в Docker компьютера, затем явно загружаем в minikube. Команда `docker-env -u` убирает переключение на Docker minikube, если оно осталось в текущем терминале от предыдущих попыток.

В выводе проверки должен быть образ `filimonq/wishlist-service:k8s-v1`. Если его нет или сборка завершилась ошибкой, дальше не переходите. В манифестах указан `imagePullPolicy: Never`, поэтому Kubernetes не скачает отсутствующий образ самостоятельно.

## 2. Секреты

```bash
test -f k8s/secret.local.yaml || cp k8s/secret.example.yaml k8s/secret.local.yaml
openssl rand -hex 32
```

Откройте `secret.local.yaml` и замените `CHANGE_ME_DB_PASSWORD` в двух местах одним паролем: в `POSTGRES_PASSWORD` и в строке `DB_CONN`. Для пароля можно использовать полученное hex-значение. Повторным вызовом `openssl rand -hex 32` получите отдельный JWT-секрет и замените `CHANGE_ME_JWT_SECRET`.

`secret.local.yaml` исключён из Git, папка `k8s/` исключена из Docker-образа. В репозитории хранится только шаблон.

## 3. PostgreSQL

```bash
kubectl --context=sre apply -f k8s/namespace.yaml
kubectl --context=sre apply -f k8s/configmap.yaml
kubectl --context=sre apply -f k8s/secret.local.yaml
kubectl --context=sre apply -f k8s/postgres.yaml
kubectl --context=sre -n wishlist rollout status statefulset/postgres --timeout=180s
```

PostgreSQL запускается в одной реплике. PVC создаётся из `volumeClaimTemplates` с использованием StorageClass `standard` в minikube. Данные монтируются в `/var/lib/postgresql`.

## 4. Миграции

```bash
kubectl --context=sre apply -f k8s/migrate.yaml
kubectl --context=sre -n wishlist wait --for=condition=complete job/wishlist-migrate --timeout=150s
kubectl --context=sre -n wishlist logs job/wishlist-migrate
```

Продолжайте к запуску приложения только после сообщения `condition met`. Если ожидание затянулось, в другом терминале посмотрите Pod и события:

```bash
kubectl --context=sre -n wishlist get pods
kubectl --context=sre -n wishlist get events --sort-by=.metadata.creationTimestamp
```

Для повторного запуска миграций:

```bash
kubectl --context=sre -n wishlist delete job wishlist-migrate
kubectl --context=sre apply -f k8s/migrate.yaml
kubectl --context=sre -n wishlist wait --for=condition=complete job/wishlist-migrate --timeout=150s
kubectl --context=sre -n wishlist logs job/wishlist-migrate
```

## 5. Приложение

```bash
kubectl --context=sre apply -f k8s/app.yaml
kubectl --context=sre -n wishlist rollout status deployment/wishlist-service --timeout=120s
kubectl --context=sre -n wishlist get pods,services,pvc,jobs
minikube -p sre service wishlist-service -n wishlist --url
```

## 6. Масштабирование

```bash
kubectl --context=sre -n wishlist scale deployment/wishlist-service --replicas=3
kubectl --context=sre -n wishlist rollout status deployment/wishlist-service --timeout=120s
kubectl --context=sre -n wishlist get pods -o wide
kubectl --context=sre -n wishlist get endpointslices -l kubernetes.io/service-name=wishlist-service
```

## Диагностика

```bash
kubectl --context=sre -n wishlist logs deployment/wishlist-service
kubectl --context=sre -n wishlist logs statefulset/postgres
kubectl --context=sre -n wishlist get events --sort-by=.metadata.creationTimestamp
```
