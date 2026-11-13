# Алерты Wishlist

Команды — из корня `wishlist-service`. Здесь алерты на метрики: vmalert проверяет правила, Alertmanager отправляет уведомления в Telegram. Оба запускаются в namespace `monitoring`.

```text
приложение → vmagent → VictoriaMetrics ← vmalert → Alertmanager → Telegram
```

Алерт на логах из прошлой домашки остаётся отдельно.

Результаты проверки и скриншоты — в [отчёте](Отчёт.md).

## Условия

| Алерт | Условие | Ожидание |
|---|---|---|
| `WishlistCreationFailures` | Не менее 3 внутренних ошибок создания вишлиста за 5 минут | 1 минута |
| `GiftCreationFailures` | Не менее 3 внутренних ошибок создания подарка за 5 минут | 1 минута |
| `GiftReservationFailures` | Не менее 3 внутренних ошибок бронирования за 5 минут | 1 минута |
| `MissingGiftReservations` | Не менее 5 отказов бронирования с причиной `item_not_found` за 5 минут | 1 минута |
| `WishlistHighGoroutines` | Более 100 горутин у конкретного Pod | 2 минуты |
| `WishlistHighHeap` | Более 64 MiB выделенной Go heap у конкретного Pod | 2 минуты |

Недоступность пользовательского сценария и отсутствие подтверждения доступности PostgreSQL — `critical`. Остальные правила — `warning`. Оба уровня отправляются в Telegram; уровень указан в сообщении и участвует в группировке. Пороги выбраны для учебного стенда. Трафика мало, поэтому для ошибок взято количество за 5 минут: процент мог бы сильно вырасти от одной неудачной операции.

В первые три правила попадают только внутренние ошибки. Неверная дата, отсутствие прав и уже забронированный подарок считаются отдельно. Повторные попытки брони отсутствующего подарка — повод проверить ссылки и клиент.

Бизнес-счётчики складываются по репликам через `sum(increase(...))`. Результат может быть дробным из-за интервалов сбора. Горутины и heap проверяются у каждого Pod отдельно, при `up=1`. Heap — только часть памяти Go-процесса.

### Доступность и SLO

| Алерт | Условие | Ожидание |
|---|---|---|
| `WishlistUnavailable` | Работающий пробер получает неуспешный результат чтения вишлиста | 1 минута |
| `WishlistPostgresUnavailable` | Нет свежего подтверждения `pg_up=1` при `up=1`, включая исчезновение метрик | 1 минута |
| `WishlistProberUnhealthy` | Нет успешного сбора метрик пробера или завершённой проверки за минуту | 1 минута |
| `WishlistMetricsUnavailable` | Ни одна реплика backend не предоставляет метрики, включая исчезновение всех рядов `up` | 1 минута |
| `WishlistSuccessSLOBreach` | Менее 99,9% API-запросов без `5xx` за 5 минут | 2 минуты |
| `WishlistLatencySLOBreach` | Менее 99% API-запросов завершились за 500 мс за 5 минут | 2 минуты |

Проверки доступности берут последние значения не старше минуты. После исчезновения метрик сначала истекает это окно, затем идёт ожидание `for`; доставка добавляет время сбора и группировки. Проверка PostgreSQL обнаруживает и отказ БД, и потерю её наблюдения — причину нужно уточнять по exporter и vmagent.

Если пробер жив, но проверка зависла, это обнаруживается по timestamp последней завершённой проверки. Когда сам пробер недоступен, правило пользовательского сценария не делает выводов по старому результату. При полном отказе backend исчезновение его метрик обнаруживается независимо от пробера. Отказ одной реплики при работающих остальных это правило не считает полным отказом.

Правила SLO используют показатели и цели из [предыдущей работы](../slo/Отчёт.md). Это оперативные сигналы по окну 5 минут, а не доказательство нарушения недельного SLO или SLA. Health checks исключены по маршруту. При отсутствии трафика правила SLO не срабатывают; доступность проверяет пробер. При малом трафике даже одна ошибка может пересечь порог.

Это проверки внутри действующей системы мониторинга. При остановке самих vmalert или Alertmanager уведомления не придут. При отказе VictoriaMetrics выражения могут завершаться ошибкой, а не алертом об отсутствующих метриках. Для контроля всей цепочки нужен независимый мониторинг.

Диагностика: [сбор метрик](../metrics/README.md) и [состояние пробера](../slo/README.md#проверка-пробера).

## Telegram

1. В `@BotFather` отправить `/newbot`, задать имя и username.
2. Открыть своего бота и отправить `/start`.
3. В терминале:

```bash
make alerting-config
```

Вставить токен в скрытый ввод. Команда сама найдёт ID чата. Если чатов несколько — предложит выбрать. Бот должен быть отдельным, без другого обработчика сообщений.

Настройки сохраняются в `alertmanager.local.yaml` с правами `0600`. Файл исключён из Git; в архив его тоже не класть. Для обновления токена запустить команду ещё раз. В репозитории остаётся только шаблон `alertmanager.example.yaml`.

Первое уведомление отправляется после 10 секунд группировки, изменения группы — раз в 30 секунд. Если алерт продолжает гореть, повтор — раз в 4 часа. После восстановления тоже приходит сообщение.

## Запуск

Сначала должны работать приложение, PostgreSQL, VictoriaMetrics и vmagent — [команды здесь](../metrics/README.md). Namespace `monitoring` уже создаётся при установке VictoriaMetrics.

Собрать backend с новыми счётчиками и обновить Deployment:

```bash
eval "$(minikube -p sre docker-env -u)"
docker build -t filimonq/wishlist-service:alerts-v1 .
minikube -p sre image load --daemon=true filimonq/wishlist-service:alerts-v1
kubectl --context=sre -n wishlist set image deployment/wishlist-service wishlist-service=filimonq/wishlist-service:alerts-v1
kubectl --context=sre -n wishlist set env deployment/wishlist-service APP_VERSION=alerts-v1
kubectl --context=sre -n wishlist rollout status deployment/wishlist-service --timeout=180s
```

В `k8s/app.yaml` пока указан `slo-v1`. Если применить его снова, образ вернётся на старый — после этого повторить обновление выше. Для следующей сборки изменённого кода взять новый тег.

После `make alerting-config`:

```bash
kubectl --context=sre apply -f k8s/alerting/alertmanager.local.yaml
kubectl --context=sre apply -f k8s/alerting/alertmanager.yaml
kubectl --context=sre -n monitoring rollout status deployment/wishlist-alertmanager --timeout=300s
kubectl --context=sre apply -f k8s/alerting/rules.yaml
kubectl --context=sre apply -f k8s/alerting/vmalert.yaml
kubectl --context=sre -n monitoring rollout status deployment/wishlist-metrics-vmalert --timeout=180s
```

Применять именно эти файлы. Во всей папке есть ещё шаблон Secret с заглушками.

В отдельных терминалах:

```bash
kubectl --context=sre -n monitoring port-forward service/wishlist-metrics-vmalert 8881:8880
```

```bash
kubectl --context=sre -n monitoring port-forward service/wishlist-alertmanager 9093:9093
```

- [Правила vmalert](http://localhost:8881/vmalert/groups)
- [Alertmanager](http://localhost:9093/)
- [Метрики vmui](http://localhost:8428/vmui/) — нужен отдельный port-forward VictoriaMetrics на `8428`.

У правил должны быть `health: ok` и пустой `lastError`.

- `Inactive` — условие не выполняется.
- `Pending` — порог превышен, идёт отсчёт `for`.
- `Firing` — условие держалось достаточно долго, алерт отправляется в Alertmanager.

После изменения локальных настроек Telegram:

```bash
kubectl --context=sre apply -f k8s/alerting/alertmanager.local.yaml
kubectl --context=sre -n monitoring rollout restart deployment/wishlist-alertmanager
kubectl --context=sre -n monitoring rollout status deployment/wishlist-alertmanager --timeout=180s
```

После изменения правил:

```bash
kubectl --context=sre apply -f k8s/alerting/rules.yaml
kubectl --context=sre -n monitoring rollout restart deployment/wishlist-metrics-vmalert
```

## Проверка

Для новых правил:

- Остановить backend через масштабирование до нуля: работающий пробер обнаружит отказ сценария, отдельно исчезнут метрики backend. Вернуть исходное число реплик.
- Остановить PostgreSQL: исчезновение `pg_up` тоже должно вызвать алерт. Вернуть одну реплику и дождаться готовности.
- Остановить пробер: должен сработать `WishlistProberUnhealthy`. Вернуть одну реплику.
- Для SLO оставить пробер работающим, выставить у backend `DEMO_ERROR_RATE=1` либо `DEMO_DELAY=1s`. Поддерживать условие не меньше 2 минут после пересечения порога. Затем вернуть `DEMO_ERROR_RATE=0 DEMO_DELAY=0s`.

При восстановлении данные об ошибках и задержках остаются в пятиминутном окне. Число реплик перед экспериментами записать, после — восстановить. Новые проверки на стенде ещё нужно выполнить; старые скриншоты в отчёте относятся к исходным шести правилам.

Для `MissingGiftReservations` нужны несколько запросов бронирования: токен существующего вишлиста и UUID отсутствующего подарка. Сам UUID должен быть корректным, иначе запрос отсеется раньше и в бизнес-счётчик не попадёт. Перед запросами дать vmagent хотя бы 30 секунд собрать исходные значения.

Для проверки runtime временно снизить пороги:

```bash
kubectl --context=sre -n monitoring set env deployment/wishlist-metrics-vmalert GOROUTINES_THRESHOLD=1 HEAP_BYTES_THRESHOLD=1
kubectl --context=sre -n monitoring rollout status deployment/wishlist-metrics-vmalert --timeout=180s
```

Подождать 2 минуты плюс время отправки сообщения. После замены Pod может понадобиться заново запустить port-forward. Это проверка уведомлений со сниженным порогом; в отчёте так и указать.

Вернуть обычные значения:

```bash
kubectl --context=sre -n monitoring set env deployment/wishlist-metrics-vmalert GOROUTINES_THRESHOLD=100 HEAP_BYTES_THRESHOLD=67108864
kubectl --context=sre -n monitoring rollout status deployment/wishlist-metrics-vmalert --timeout=180s
```

### Внутренние ошибки бизнес-операций

Для этих трёх правил отключаем БД и обращаемся прямо к Pod. Через обычный Service запросы не дойдут: readiness уберёт backend из готовых endpoints. Старый `DEMO_ERROR_RATE` здесь не подходит — он меняет только публичное чтение.

Найти backend Pod, подставить его имя вместо `POD_NAME` и оставить туннель в отдельном терминале:

```bash
kubectl --context=sre -n wishlist get pods -l app=wishlist-service
kubectl --context=sre -n wishlist port-forward pod/POD_NAME 18080:8080
```

На `http://localhost:18080/` войти, создать тестовый вишлист с подарком, открыть публичную страницу в другой вкладке. Подождать 30 секунд. Посмотреть число реплик БД, запомнить его и выключить PostgreSQL:

```bash
kubectl --context=sre -n wishlist get statefulset postgres -o jsonpath='{.spec.replicas}{"\n"}'
kubectl --context=sre -n wishlist scale statefulset/postgres --replicas=0
kubectl --context=sre -n wishlist wait --for=delete pod/postgres-0 --timeout=90s
```

В подготовленных вкладках сделать по 6 попыток создания вишлиста, добавления подарка и бронирования. Данные должны быть корректными: будущая дата, непустое название, приоритет 1–5. Должны вернуться `500`, а счётчики `internal_error` — вырасти. Метрики с Pod по-прежнему собираются.

Сразу вернуть исходное число реплик БД. Здесь — вариант с одной репликой:

```bash
kubectl --context=sre -n wishlist scale statefulset/postgres --replicas=1
kubectl --context=sre -n wishlist rollout status statefulset/postgres --timeout=180s
kubectl --context=sre -n wishlist rollout status deployment/wishlist-service --timeout=180s
```

Ошибки ещё до 5 минут остаются в окне расчёта, поэтому алерт может гореть и после восстановления БД. Когда условие перестанет выполняться, придёт сообщение о восстановлении. Перезапуск vmalert начинает отсчёт `for` заново.

Для сдачи сохранить скриншоты правил и сообщений в Telegram. В отчёте кратко описать условия и способ проверки, включая временное снижение порогов.

Если что-то не работает:

```bash
kubectl --context=sre -n monitoring logs deployment/wishlist-metrics-vmalert --tail=50
kubectl --context=sre -n monitoring logs deployment/wishlist-alertmanager --tail=50
kubectl --context=sre -n monitoring get pods,pvc
```

Документация: [vmalert](https://docs.victoriametrics.com/victoriametrics/vmalert/), [Alertmanager](https://prometheus.io/docs/alerting/latest/configuration/).
