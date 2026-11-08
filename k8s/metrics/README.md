# Метрики в minikube

Команды — из корня `wishlist-service`. Кластер — `sre`, мониторинг — в namespace `monitoring`.

| Файл | Назначение |
|---|---|
| `victoria-metrics-values.yaml` | Настройки сервера хранения VictoriaMetrics |
| `vmagent-values.yaml` | Настройки сборщика и обнаружения подов |
| `vmagent-rbac.yaml` | Права сборщика на чтение подов в namespace `wishlist` |

Сначала нужно [поднять приложение](../README.md) с актуальным образом и манифестами. Дальше — установить VictoriaMetrics и vmagent по командам ниже.

## VictoriaMetrics

VictoriaMetrics хранит метрики. Здесь один экземпляр, диск 2 GiB, хранение 7 дней, порт `8428`.

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

У Pod должно быть `1/1 Running`, у PVC — `Bound`. Если Helm не дождался запуска, посмотреть причину:

```bash
kubectl --context=sre -n monitoring get pods,pvc
kubectl --context=sre -n monitoring get events --sort-by=.metadata.creationTimestamp
kubectl --context=sre -n monitoring logs statefulset/victoria-metrics --tail=50
```

### 3. Проверка и UI

В отдельном терминале:

```bash
kubectl --context=sre -n monitoring port-forward service/victoria-metrics 8428:8428
```

UI — [localhost:8428/vmui/](http://localhost:8428/vmui/). Проверить, что сервер отвечает:

```bash
curl --fail --show-error http://localhost:8428/health
```

Метрики появятся после запуска vmagent: VictoriaMetrics сама их у приложения не забирает.

### Остановка

```bash
helm --kube-context sre uninstall vm-single --namespace monitoring
```

PVC StatefulSet сохраняется после удаления релиза. Остановка minikube командой `minikube stop -p sre` также сохраняет данные.

Основа конфигурации: [официальная инструкция Helm-чарта](https://docs.victoriametrics.com/helm/victoria-metrics-single/).

## postgres-exporter

В Pod PostgreSQL есть второй контейнер — [postgres-exporter](https://github.com/prometheus-community/postgres_exporter/tree/v0.20.0) `v0.20.0`. Он читает статистику БД через `localhost:5432` и отдаёт её на `:9187/metrics`.

Пользователь и имя БД берутся из ConfigMap `wishlist-config`, пароль — из Secret `wishlist-secrets`.

Запуск или обновление:

```bash
kubectl --context=sre apply -f k8s/postgres.yaml
kubectl --context=sre -n wishlist rollout status statefulset/postgres --timeout=180s
kubectl --context=sre -n wishlist get pods -l app=wishlist-postgres
kubectl --context=sre -n wishlist logs statefulset/postgres -c postgres-exporter --tail=50
```

У `postgres-0` должно быть `READY 2/2`. Чтобы посмотреть исходные метрики, в отдельном терминале:

```bash
kubectl --context=sre -n wishlist port-forward pod/postgres-0 9187:9187
```

В другом терминале:

```bash
curl --fail --show-error http://localhost:9187/metrics
```

В ответе должны быть `pg_up 1` и статистика БД, например `pg_stat_database_xact_commit`.

Когда vmagent уже работает, в [vmui](http://localhost:8428/vmui/) можно проверить два показателя. Запускать по одному:

```promql
last_over_time(up{namespace="wishlist", app="wishlist-postgres"}[30s])
```

```promql
last_over_time(pg_up{namespace="wishlist", app="wishlist-postgres"}[30s])
```

`up=1` — vmagent забрал метрики экспортера. `pg_up=1` — экспортер смог получить данные из PostgreSQL. Если `up=1`, а `pg_up=0`, смотреть подключение к БД и логи экспортера. Если результат пустой, свежих данных за 30 секунд нет.

TCP-пробы проверяют только порт экспортера. Это не проверка подключения к БД. Порт `9187` наружу не опубликован — vmagent ходит прямо на IP Pod.

## vmagent и service discovery

vmagent раз в 15 секунд делает HTTP-запросы к `/metrics` приложения и postgres-exporter, затем отправляет полученные значения в VictoriaMetrics:

```text
wishlist-service:9090/metrics ─┐
                             ├─ vmagent ── VictoriaMetrics:8428
postgres-exporter:9187/metrics ┘
```

vmagent находит Pods через Kubernetes API. В namespace `wishlist` он выбирает Running Pods с `prometheus.io/scrape: "true"`. Порт и путь берёт из аннотаций `prometheus.io/port` и `prometheus.io/path`. Аннотации записаны в шаблонах Pods, поэтому новые реплики тоже обнаруживаются автоматически.

Фильтр оставляет нужный порт: `9090` у приложения, `9187` у экспортера. Каждая реплика собирается отдельно. К метрикам добавляются labels `namespace`, `pod`, `container` и `app`.

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

Открыть [localhost:8429/targets](http://localhost:8429/targets). Там должна быть каждая реплика приложения и postgres-exporter со статусом `UP`. После масштабирования дать сборщику время найти новые Pods.

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
wishlist_items_created_total{app="wishlist-service"}
wishlist_items_reserved_total{app="wishlist-service"}
pg_stat_database_xact_commit{app="wishlist-postgres"}
```

Выполняйте запросы по одному. После действий в UI приложения дождитесь следующего сбора и проверьте рост соответствующих счётчиков.

### Ошибки бизнес-операций и несколько реплик

`wishlist_user_login_failures_total{reason}` считает неудачные входы: `invalid_credentials` и `internal_error`.

`wishlist_item_reservation_failures_total{reason}` считает отказы бронирования: `wishlist_not_found`, `item_not_found`, `item_not_in_wishlist`, `already_reserved`, `internal_error`.

Одна неудачная операция — одно увеличение счётчика. Неверный JWT, JSON или UUID, отклонённые раньше вызова сервиса, видны в HTTP-метриках. ID пользователей и тексты ошибок в labels не попадают.

`wishlist_items_created_total` — успешно добавленные подарки. По нему и числу созданных вишлистов можно смотреть общую активность. Чтобы понять, кто создал вишлист и ушёл без подарков, нужны связанные события пользователей.

Ошибки создания: `wishlist_wishlist_creation_failures_total{reason}` (`invalid_event_date`, `internal_error`) и `wishlist_item_creation_failures_total{reason}` (`invalid_priority`, `wishlist_not_found`, `access_denied`, `internal_error`). Алерты на них — в [отдельной инструкции](../alerting/README.md).

У каждой реплики свой Counter, после перезапуска он обнуляется. Число событий за 5 минут по всем репликам:

```promql
sum(increase(wishlist_items_reserved_total{namespace="wishlist", app="wishlist-service"}[5m]))
```

Число ошибок по причинам за тот же период:

```promql
sum by (reason) (increase(wishlist_user_login_failures_total{namespace="wishlist", app="wishlist-service"}[5m]))
```

```promql
sum by (reason) (increase(wishlist_item_reservation_failures_total{namespace="wishlist", app="wishlist-service"}[5m]))
```

Для расчёта нужны значения до и после события. Пустой результат не означает, что ошибок нет. Проверка самого пробера — в [инструкции SLO](../slo/README.md#проверка-пробера).

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
