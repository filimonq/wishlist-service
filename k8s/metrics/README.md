# Метрики в minikube

Команды выполняются из корня `wishlist-service`. Профиль minikube — `sre`, namespace мониторинга — `monitoring`.

| Файл | Назначение |
|---|---|
| `victoria-metrics-values.yaml` | Настройки сервера хранения VictoriaMetrics |
| `vmagent-values.yaml` | Настройки сборщика и обнаружения подов |
| `vmagent-rbac.yaml` | Права сборщика на чтение подов в namespace `wishlist` |

Для общего запуска сначала пересоберите и загрузите образ приложения, затем выполните шаги запуска из [основной инструкции](../README.md). Примените актуальные `configmap.yaml`, `postgres.yaml` и `app.yaml`: в них добавлены порт метрик, экспортер БД и аннотации. Старый образ без нового Go-кода метрики не отдаст. Затем установите VictoriaMetrics и vmagent по разделам ниже.

## VictoriaMetrics

VictoriaMetrics хранит временные ряды. Здесь запускается один экземпляр с диском 2 GiB и сроком хранения 7 дней. Сервис доступен внутри кластера на порту `8428`, с компьютера — через `port-forward`.

Используется официальный чарт `victoria-metrics-single` версии `0.49.0`. Версия VictoriaMetrics задаётся самим чартом — `v1.154.0`. Настройки для minikube находятся в `victoria-metrics-values.yaml`.

### 1. Подготовка

Нужны Docker, minikube, kubectl и [Helm](https://helm.sh/docs/intro/install/).

```bash
helm version --short
minikube start -p sre --driver=docker --container-runtime=docker
kubectl --context=sre get nodes
kubectl --context=sre get storageclass standard

helm repo add vm https://victoriametrics.github.io/helm-charts/
helm repo update vm
```

### 2. Установка

```bash
helm upgrade --install vm-single vm/victoria-metrics-single \
  --version 0.49.0 \
  --kube-context sre \
  --namespace monitoring --create-namespace \
  --values k8s/metrics/victoria-metrics-values.yaml \
  --wait --timeout 5m

kubectl --context=sre -n monitoring get pods,services,pvc
```

Дождитесь `Running` и `1/1` у Pod, `Bound` у PVC. Первый запуск может занять время из-за скачивания образа. Если установка завершилась по таймауту:

```bash
kubectl --context=sre -n monitoring get pods,pvc
kubectl --context=sre -n monitoring get events --sort-by=.metadata.creationTimestamp
kubectl --context=sre -n monitoring logs statefulset/victoria-metrics --tail=50
```

### 3. Проверка и UI

В отдельном терминале оставьте работающим:

```bash
kubectl --context=sre -n monitoring port-forward service/victoria-metrics 8428:8428
```

Откройте [http://localhost:8428/vmui/](http://localhost:8428/vmui/). Проверить доступность можно из другого терминала:

```bash
curl --fail --show-error http://localhost:8428/health
```

До запуска vmagent метрики приложения в UI отсутствуют: сам по себе сервер хранения их не забирает.

### Остановка

```bash
helm --kube-context sre uninstall vm-single --namespace monitoring
```

PVC StatefulSet сохраняется после удаления релиза. Остановка minikube командой `minikube stop -p sre` также сохраняет данные.

Основа конфигурации: [официальная инструкция Helm-чарта](https://docs.victoriametrics.com/helm/victoria-metrics-single/).

## postgres-exporter

В `k8s/postgres.yaml` добавлен второй контейнер с [postgres-exporter](https://github.com/prometheus-community/postgres_exporter/tree/v0.20.0) версии `v0.20.0`. Он подключается к PostgreSQL через `localhost:5432` и отдаёт статистику БД на `:9187/metrics`. Диск PostgreSQL экспортеру не монтируется.

Для подключения используется текущий пользователь БД из ConfigMap `wishlist-config`, пароль берётся из Secret `wishlist-secrets`. Они должны быть созданы по [основной инструкции](../README.md). Переменная `POSTGRES_DB` объявлена перед `DATA_SOURCE_URI`, чтобы Kubernetes подставил имя БД в адрес подключения.

При общем запуске примените обновлённый StatefulSet:

```bash
kubectl --context=sre apply -f k8s/postgres.yaml
kubectl --context=sre -n wishlist rollout status statefulset/postgres --timeout=180s
kubectl --context=sre -n wishlist get pods -l app=wishlist-postgres
kubectl --context=sre -n wishlist logs statefulset/postgres -c postgres-exporter --tail=50
```

У Pod `postgres-0` ожидается `READY 2/2`. Для проверки метрик оставьте в отдельном терминале:

```bash
kubectl --context=sre -n wishlist port-forward pod/postgres-0 9187:9187
```

В другом терминале:

```bash
curl --fail --show-error http://localhost:9187/metrics
```

В выдаче должны быть `pg_up 1`, `pg_stat_database_xact_commit` и другие метрики PostgreSQL. `pg_up 0` означает, что экспортер доступен, но не смог подключиться к БД. Его probes проверяют только доступность HTTP-порта; состояние соединения с PostgreSQL проверяется по `pg_up`.

Порт `9187` не публикуется через NodePort. Метрики забирает vmagent напрямую с IP пода.

## vmagent и service discovery

vmagent раз в 15 секунд делает HTTP-запросы к `/metrics` приложения и postgres-exporter, затем отправляет полученные значения в VictoriaMetrics:

```text
wishlist-service:9090/metrics ─┐
                             ├─ vmagent ── VictoriaMetrics:8428
postgres-exporter:9187/metrics ┘
```

Service discovery — автоматическое обнаружение целей сбора через Kubernetes API. vmagent следит за подами в namespace `wishlist` и выбирает запущенные поды с аннотацией `prometheus.io/scrape: "true"`. Аннотации `prometheus.io/port` и `prometheus.io/path` указывают порт и путь метрик. Они находятся в `spec.template.metadata.annotations` у Deployment приложения и StatefulSet PostgreSQL, поэтому автоматически попадают на каждый новый Pod.

Фильтр оставляет только порт контейнера, совпадающий с аннотацией: `9090` у приложения и `9187` у экспортера. Это исключает повторный сбор с одного Pod и запросы метрик к порту БД `5432` или API `8080`. Каждая реплика приложения собирается отдельно; после масштабирования vmagent сам обнаружит новые поды. В метрики добавляются лейблы `namespace`, `pod`, `container` и `app`.

`Role` и `RoleBinding` разрешают ServiceAccount `monitoring/vmagent` читать поды только в `wishlist`. Доступ к Secret и права на изменение ресурсов ему не выдаются.

Используется официальный чарт `victoria-metrics-agent` версии `0.50.0` с vmagent `v1.154.0`. Очередь отправки ограничена 128 MB и размещается в `emptyDir`: при пересоздании Pod неотправленные данные теряются. Уже принятые VictoriaMetrics данные хранятся на её PVC.

### Установка

После запуска приложения и VictoriaMetrics:

```bash
kubectl --context=sre apply -f k8s/metrics/vmagent-rbac.yaml

helm upgrade --install vmagent vm/victoria-metrics-agent \
  --version 0.50.0 \
  --kube-context sre \
  --namespace monitoring \
  --values k8s/metrics/vmagent-values.yaml \
  --wait --timeout 5m

kubectl --context=sre -n monitoring get pods,services,pvc
```

### Проверка сбора

В отдельном терминале:

```bash
kubectl --context=sre -n monitoring port-forward service/vmagent 8429:8429
```

Откройте [http://localhost:8429/targets](http://localhost:8429/targets). Ожидается по одной цели для каждой реплики приложения и одна для postgres-exporter. У всех состояние `UP`. Обнаружение новых подов и первый сбор могут занять некоторое время.

При работающем `port-forward` VictoriaMetrics на `8428`:

```bash
curl --fail --show-error --get http://localhost:8428/api/v1/query \
  --data-urlencode 'query=up{namespace="wishlist"}'

curl --fail --show-error --get http://localhost:8428/api/v1/query \
  --data-urlencode 'query=pg_up{namespace="wishlist"}'
```

Значение `up = 1` означает успешный сбор HTTP-эндпоинта. `pg_up = 1` дополнительно подтверждает, что postgres-exporter подключился к PostgreSQL.

В [vmui](http://localhost:8428/vmui/) можно выполнить:

```promql
go_goroutines{app="wishlist-service"}
wishlist_user_registrations_total{app="wishlist-service"}
wishlist_user_logins_total{app="wishlist-service"}
wishlist_wishlists_created_total{app="wishlist-service"}
wishlist_items_reserved_total{app="wishlist-service"}
pg_stat_database_xact_commit{app="wishlist-postgres"}
```

Выполняйте запросы по одному. После действий в UI приложения дождитесь следующего сбора и проверьте рост соответствующих счётчиков.

Для диагностики:

```bash
kubectl --context=sre -n monitoring logs deployment/vmagent --tail=50
```

### Удаление сборщика

```bash
helm --kube-context sre uninstall vmagent --namespace monitoring
kubectl --context=sre delete -f k8s/metrics/vmagent-rbac.yaml
```

Основа конфигурации: [официальный Helm-чарт vmagent](https://docs.victoriametrics.com/helm/victoria-metrics-agent/).
