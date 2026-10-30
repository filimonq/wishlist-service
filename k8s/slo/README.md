# SLI, SLO и SLA

Определения и цели находятся в [Отчёт.md](Отчёт.md), запросы для графиков — в [queries.md](queries.md). Здесь описаны запуск пробера и управление демонстрационными сбоями.

## Пробер

`cmd/prober` — отдельная Go-команда в том же образе, что и приложение. Она раз в 5 секунд читает существующий вишлист через публичный API. Успех — HTTP `200`, корректный JSON и совпадение поля `id` с ожидаемым UUID. Таймаут всей проверки, включая чтение ответа, — 2 секунды. Редиректы не считаются успехом.

Каждая проверка открывает новое соединение к Service, чтобы не закрепляться за одной репликой через keep-alive. URL, токен вишлиста и его ID не добавляются в лейблы или логи. Ответ ограничен размером 1 MiB.

Проверки запланированы на границах пятисекундных интервалов. Хорошая UTC-минута содержит все 12 успешных проверок. Неполная первая минута, пропущенные проверки и минуты с ошибками считаются плохими. Пробер работает в одной реплике со стратегией `Recreate`, чтобы не удваивать проверки при обновлении.

### Конфигурация

| Переменная | Значение |
|---|---|
| `PROBE_URL` | Полный URL публичного API вишлиста, обязателен |
| `PROBE_EXPECTED_WISHLIST_ID` | UUID ожидаемого вишлиста, обязателен |
| `PROBE_METRICS_PORT` | Порт метрик пробера, по умолчанию `9115` |

При неправильной конфигурации команда завершится с ошибкой. Неуспешный запрос к приложению оставляет пробер работающим: отказ должен попасть в метрики, а не вызвать перезапуск самого наблюдателя.

### Метрики

| Метрика | Значение |
|---|---|
| `wishlist_probe_checks_total{result=...}` | Число проверок: `success`, `http_error`, `invalid_response`, `network_error`, `timeout`, `missed` |
| `wishlist_probe_success` | Результат последней проверки: `1` или `0` |
| `wishlist_probe_duration_seconds` | Длительность последней проверки |
| `wishlist_probe_last_check_timestamp_seconds` | Время завершения последней проверки |
| `wishlist_probe_minutes_total{result="good\|bad"}` | Счётчики завершённых хороших и плохих минут |
| `wishlist_probe_last_minute_success` | Результат последней завершённой минуты: `1` или `0` |
| `wishlist_probe_last_minute_timestamp_seconds` | Время конца этой минуты |

До первой проверки или первой завершённой минуты соответствующие timestamp равны `0`: результата ещё нет. После перезапуска счётчики обнуляются. Если пробер отсутствует, это нужно учитывать по свежести метрик и ожидаемому числу минут, а не исключать пропуски из SLA.

### Локально

Сначала создайте вишлист в UI или через API. Из ответа API возьмите `id` и `public_token`.

```bash
PROBE_URL="http://localhost:8080/api/v1/public/<public_token>" \
PROBE_EXPECTED_WISHLIST_ID="<id>" \
go run ./cmd/prober
```

В другом терминале:

```bash
curl --fail --show-error http://localhost:9115/metrics
```

### В minikube

Перед запуском соберите образ с отдельным тегом и загрузите его именно из Docker-демона компьютера:

```bash
eval "$(minikube -p sre docker-env -u)"
docker build -t filimonq/wishlist-service:slo-v1 .
minikube -p sre image load --daemon=true filimonq/wishlist-service:slo-v1
```

Манифесты приложения, миграций и пробера используют `slo-v1`. Старый образ не содержит `/app/wishlist-prober` и demo-middleware. Приложение и мониторинг запускаются по [инструкции по метрикам](../metrics/README.md).

Создайте вишлист и заполните локальную копию манифеста:

```bash
cp -n k8s/slo/prober.example.yaml k8s/slo/prober.local.yaml
```

В `prober.local.yaml` замените `CHANGE_ME_PUBLIC_TOKEN` на `public_token`, а `CHANGE_ME_WISHLIST_ID` — на `id` одного и того же вишлиста. Если меняется порт пробера, обновите также аннотацию и `containerPort`.

```bash
kubectl --context=sre apply -f k8s/slo/prober.local.yaml
kubectl --context=sre -n wishlist rollout status deployment/wishlist-prober --timeout=180s
kubectl --context=sre -n wishlist logs deployment/wishlist-prober --tail=20
```

Уже настроенный vmagent обнаружит Pod по аннотациям. После изменения ConfigMap перезапустите пробер:

```bash
kubectl --context=sre -n wishlist rollout restart deployment/wishlist-prober
```

Остановка — `Ctrl+C` локально или удаление Deployment `wishlist-prober` в Kubernetes. SIGTERM отменяет текущий запрос и завершает сервер метрик.

## Управляемые ошибки и задержки

Пакет `internal/adapter/in/http/demo` содержит отдельный middleware только для `GET /api/v1/public/:token`. Он выполняется внутри измерения HTTP-метрик, перед чтением вишлиста. Обработчики бизнес-операций и репозитории не содержат демонстрационных условий.

| Переменная | Назначение | По умолчанию |
|---|---|---|
| `DEMO_DELAY` | Задержка перед обработкой запроса, например `1s` | `0s` |
| `DEMO_ERROR_RATE` | Вероятность ответа `503`: от `0` до `1` | `0` |

Задержка применяется первой, затем выбирается, вернуть `503` или вызвать обычный обработчик. При отмене запроса ожидание прерывается; обработчик вишлиста не вызывается. Некорректные значения переменных блокируют запуск HTTP-приложения. Настройки считываются при старте.

Для воспроизводимой демонстрации используются отдельные режимы. Команды выполняются после общего запуска окружения.

Задержка в 1 секунду: превышает цель 500 мс, но укладывается в таймаут пробера 2 секунды:

```bash
kubectl --context=sre -n wishlist set env deployment/wishlist-service DEMO_DELAY=1s DEMO_ERROR_RATE=0
kubectl --context=sre -n wishlist rollout status deployment/wishlist-service --timeout=180s
```

Ошибки: каждый запрос публичного чтения получает `503`, поэтому ухудшаются серверная успешность и результат пробера:

```bash
kubectl --context=sre -n wishlist set env deployment/wishlist-service DEMO_DELAY=0s DEMO_ERROR_RATE=1
kubectl --context=sre -n wishlist rollout status deployment/wishlist-service --timeout=180s
```

Восстановление обычной работы:

```bash
kubectl --context=sre -n wishlist set env deployment/wishlist-service DEMO_DELAY=0s DEMO_ERROR_RATE=0
kubectl --context=sre -n wishlist rollout status deployment/wishlist-service --timeout=180s
```

Изменение переменных запускает rollout приложения; пробер остаётся работающим. После переключения режима нужно дождаться появления новых метрик и обновления расчётного окна. Health checks, вход, создание вишлистов и бронирование этим middleware не затрагиваются.

## Подготовка и порядок демонстрации

1. Пересоберите и загрузите образ. Поднимите приложение и мониторинг по [инструкции по метрикам](../metrics/README.md).
2. Создайте контрольный вишлист, заполните `prober.local.yaml` и запустите пробер. Его цель должна появиться на странице vmagent `/targets` со статусом `UP`.
3. Оставьте в отдельных терминалах пробросы портов:

```bash
kubectl --context=sre -n monitoring port-forward service/victoria-metrics 8428:8428
```

```bash
kubectl --context=sre -n monitoring port-forward service/vmagent 8429:8429
```

4. Откройте [vmui](http://localhost:8428/vmui/) и подготовьте графики по [queries.md](queries.md). На каждом добавьте линию соответствующей цели. Диапазон просмотра — последние 15–30 минут; шаг графика — 15 секунд, если доступна ручная настройка.
5. До записи дайте проберу поработать не менее 11 минут в обычном режиме. Проверьте успешные ответы и появление хороших минут. Неполная стартовая минута считается плохой.
6. На записи покажите исходные графики, затем включите ошибки на 1–2 минуты, верните обычную работу, включите задержку на 1–2 минуты и снова восстановите сервис. После каждого rollout дождитесь обновления метрик. Для восстановления запросных SLI нужно обновление минутного окна.
7. Отдельно покажите реакцию индикатора PostgreSQL. Остановка выполняется только в учебном окружении; данные на PVC сохраняются:

```bash
kubectl --context=sre -n wishlist scale statefulset/postgres --replicas=0
```

Подержите БД выключенной около 90 секунд, затем восстановите:

```bash
kubectl --context=sre -n wishlist scale statefulset/postgres --replicas=1
kubectl --context=sre -n wishlist rollout status statefulset/postgres --timeout=180s
kubectl --context=sre -n wishlist rollout status deployment/wishlist-service --timeout=180s
```

При остановке Pod исчезает и postgres-exporter: отсутствие его данных учитывается как плохие минутные отсчёты. Пробер продолжает фиксировать недоступность пользовательского сценария.

8. Покажите успешную текущую проверку пробера после восстановления. Накопленная доступность за 10 минут может оставаться ниже цели: произошедшие плохие минуты ещё входят в окно. Дожидаться возврата этой линии к 100% на записи не требуется.

В отчёте зафиксированы определения и реализованный код. Результаты выполнения сценария добавляются после проверки в minikube; пока они не заявлены.
