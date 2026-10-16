# Victoria Logs и Vector в minikube

Команды выполняются из корня `wishlist-service`. Используется тот же кластер `sre` и namespace `wishlist`, что и для приложения.

## Как это работает

Приложение пишет JSON в stdout. Vector читает эти записи на узле и отправляет их в Victoria Logs. Смотреть и искать логи будем через встроенный интерфейс vmui.

В `victoria-logs.yaml` описаны одна копия хранилища версии `v1.53.0`, диск на 2 ГиБ и внутренний Service `victoria-logs:9428`. Данные лежат в `/storage` и хранятся 7 дней. При обновлении старый Pod останавливается перед запуском нового, чтобы они не писали на диск одновременно.

Диск переживает замену Pod. Удаление namespace `wishlist` удаляет и заявку на этот диск, поэтому такой сброс не используем, если хотим сохранить логи.

## Подготовить приложение

Перед запуском нужен образ с новыми JSON-логами. Соберите текущий код и загрузите образ в minikube:

```bash
eval "$(minikube -p sre docker-env -u)"
docker build -t filimonq/wishlist-service:k8s-v1 .
minikube -p sre image load filimonq/wishlist-service:k8s-v1
minikube -p sre image ls | grep wishlist-service
```

Для этого локального стенда оставляем тег `k8s-v1`. Удаление namespace не удаляет образы, поэтому загрузка новой сборки обязательна.

Если приложение ещё не запущено, поднимите его по [инструкции](../README.md), разделы 2–5. Если уже работает, после загрузки образа обновите настройки и перезапустите Pods:

```bash
kubectl --context=sre apply -f k8s/configmap.yaml
kubectl --context=sre -n wishlist rollout restart deployment/wishlist-service
kubectl --context=sre -n wishlist rollout status deployment/wishlist-service --timeout=120s
kubectl --context=sre -n wishlist logs deployment/wishlist-service --tail=5
```

В логах должны быть JSON-записи с `service.version: k8s-v1` и `deployment.environment.name: minikube`. Эти значения задаются в `k8s/configmap.yaml`.

## Запуск

Кластер должен быть запущен. Первый `apply` создаст namespace, если его ещё нет.

```bash
kubectl --context=sre apply -f k8s/namespace.yaml
kubectl --context=sre apply -f k8s/logging/victoria-logs.yaml
kubectl --context=sre -n wishlist rollout status deployment/victoria-logs --timeout=180s
kubectl --context=sre -n wishlist get pods -l app=victoria-logs
kubectl --context=sre -n wishlist get service victoria-logs
kubectl --context=sre -n wishlist get pvc victoria-logs-data
```

Ожидаемые состояния: Pod — `1/1 Running`, PVC — `Bound`.

## Доступ к UI

```bash
kubectl --context=sre -n wishlist port-forward service/victoria-logs 9428:9428
```

Оставьте терминал открытым и перейдите на http://localhost:9428/select/vmui/ . `Ctrl+C` закроет доступ через localhost, само хранилище продолжит работать.

В другом терминале можно проверить HTTP-ответ:

```bash
curl --fail http://localhost:9428/health
```

Если `/health` отвечает успешно, хранилище доступно. Теперь подключаем сборщик.

## Vector

`vector.yaml` запускает Vector через DaemonSet: один сборщик на каждом Linux-узле. Здесь же лежат его конфиг и права на чтение сведений о Pods, namespaces и nodes. Читать Secret через API ему не разрешено.

Vector собирает только Pods с меткой `app=wishlist-service` в namespace `wishlist`. Он разбирает JSON, добавляет имя Pod, namespace, узла и контейнера, затем отправляет записи в Victoria Logs через совместимый с Elasticsearch bulk API.

Каталоги `/var/log/pods` и `/var/lib/docker/containers` подключены с узла только для чтения. Vector работает с UID 0, чтобы читать эти файлы. Привилегированный режим выключен, дополнительные capabilities убраны. Пути рассчитаны на наш minikube с Docker runtime.

В `/var/lib/wishlist-vector` Vector запоминает, до какого места прочитал файлы. Эти отметки сохраняются при замене его Pod. Очередь отправки пока находится в памяти: при аварии записи из неё могут потеряться.

Старые текстовые логи не проходят разбор JSON и отбрасываются; сообщения об этом видны в логах Vector.

После готовности Victoria Logs:

```bash
kubectl --context=sre apply -f k8s/logging/vector.yaml
kubectl --context=sre -n wishlist rollout status daemonset/wishlist-vector --timeout=180s
kubectl --context=sre -n wishlist get pods -l app=wishlist-vector
kubectl --context=sre -n wishlist logs daemonset/wishlist-vector --tail=50
```

Чтобы проверить доставку, откройте приложение и сделайте несколько действий. Затем выполните в vmui:

```text
_time:15m service.name:="wishlist-service" | limit 20
```

Должны появиться записи приложения. У HTTP-запросов будут поля `level`, `status`, `duration_ms` и `k8s.pod.name`. В Victoria Logs текст сообщения доступен как `_msg`, время — как `_time`. Имя сервиса, namespace и Pod определяют поток записей.

После изменения конфигурации Vector примените файл и перезапустите сборщик:

```bash
kubectl --context=sre apply -f k8s/logging/vector.yaml
kubectl --context=sre -n wishlist rollout restart daemonset/wishlist-vector
kubectl --context=sre -n wishlist rollout status daemonset/wishlist-vector --timeout=180s
```

Настройки основаны на документации [источника kubernetes_logs](https://vector.dev/docs/reference/configuration/sources/kubernetes_logs/) и [интеграции Vector с Victoria Logs](https://docs.victoriametrics.com/victorialogs/data-ingestion/vector/).

## Бонус: алерт на задержку API

`vmalert.yaml` добавляет vmalert с правилом `WishlistHighLatency`. Каждые 15 секунд он запрашивает p95 времени обработки API-запросов за последние 5 минут. Если p95 выше 500 мс непрерывно в течение минуты, алерт переходит из `Pending` в `Firing`. Пока условие не выполнено, состояние — `Inactive`.

В выборку входят маршруты `/api/v1/…`; проверки Kubernetes на `/`, frontend и Swagger исключены. Это время обработки на сервере из `duration_ms`, а не полное время запроса в браузере. 500 мс — выбранный учебный порог, а не результат измеренного SLO.

vmalert работает с Victoria Logs напрямую. Grafana, Prometheus и Alertmanager для этой проверки не нужны. Флаг `-notifier.blackhole` отключает отправку уведомлений: срабатывание видно в UI и API vmalert. Состояние хранится в памяти, после перезапуска отсчёт `for: 1m` начинается заново.

### Запуск

Приложение, Victoria Logs и Vector должны уже работать.

```bash
kubectl --context=sre apply -f k8s/logging/vmalert.yaml
kubectl --context=sre -n wishlist rollout status deployment/wishlist-vmalert --timeout=300s
kubectl --context=sre -n wishlist logs deployment/wishlist-vmalert --tail=30
```

В отдельном терминале:

```bash
kubectl --context=sre -n wishlist port-forward service/wishlist-vmalert 8880:8880
```

Откройте http://localhost:8880/ и найдите правило `WishlistHighLatency`. Проверить состояние через API можно так:

```bash
curl --fail --silent --show-error http://localhost:8880/api/v1/rules
```

### Проверка срабатывания

Для демонстрации временно снизим порог до 0 мс. Это проверка механизма алерта на реальных запросах, а не имитация аварии или доказательство высокой задержки.

```bash
kubectl --context=sre -n wishlist set env deployment/wishlist-vmalert LATENCY_THRESHOLD_MS=0
kubectl --context=sre -n wishlist rollout status deployment/wishlist-vmalert --timeout=120s
```

При замене Pod `port-forward` может завершиться. Если это произошло, запустите его снова в отдельном терминале. Откройте UI приложения и выполните несколько действий с вишлистом, чтобы в логах появились свежие API-запросы. После доставки записей и очередных проверок, обычно в течение 1–2 минут, алерт должен перейти в `Firing`. Обновляйте страницу vmalert, проверьте отсутствие ошибки выполнения правила и сделайте скриншот его состояния и условия.

Активные алерты также доступны через API:

```bash
curl --fail --silent --show-error http://localhost:8880/api/v1/alerts
```

После проверки верните порог 500 мс:

```bash
kubectl --context=sre -n wishlist set env deployment/wishlist-vmalert LATENCY_THRESHOLD_MS=500
kubectl --context=sre -n wishlist rollout status deployment/wishlist-vmalert --timeout=120s
```

При необходимости снова запустите `port-forward` и проверьте правило: если p95 ниже порога, оно должно быть `Inactive`. После правок файла с правилами примените манифест и перезапустите vmalert:

```bash
kubectl --context=sre apply -f k8s/logging/vmalert.yaml
kubectl --context=sre -n wishlist rollout restart deployment/wishlist-vmalert
kubectl --context=sre -n wishlist rollout status deployment/wishlist-vmalert --timeout=120s
```

Интеграция описана в [документации Victoria Logs](https://docs.victoriametrics.com/victorialogs/vmalert/).

## Диагностика

```bash
kubectl --context=sre -n wishlist logs deployment/victoria-logs
kubectl --context=sre -n wishlist describe pods -l app=victoria-logs
kubectl --context=sre -n wishlist logs daemonset/wishlist-vector --tail=100
kubectl --context=sre -n wishlist describe pods -l app=wishlist-vector
kubectl --context=sre -n wishlist logs deployment/wishlist-vmalert --tail=100
kubectl --context=sre -n wishlist get events --sort-by=.metadata.creationTimestamp
```
