# Grafana

Команды выполнять из корня `wishlist-service`. Нужны работающие minikube `sre`, VictoriaMetrics и vmagent из [инструкции по метрикам](../metrics/README.md).

Grafana запускается в `monitoring`. Источник `VictoriaMetrics` добавляется из `values.yaml` автоматически. Используется встроенный тип Prometheus: VictoriaMetrics поддерживает его API, отдельный плагин не нужен.

Описание панелей и скриншоты — в [отчёте](Отчёт.md).

```text
приложение / PostgreSQL / пробер → vmagent → VictoriaMetrics ← Grafana ← браузер
```

## Установка

Используется чарт `grafana-community/grafana` версии `14.4.1`. Версию образа задаёт чарт. Пароль администратора генерирует чарт и сохраняет в Kubernetes Secret; в values пароля нет.

```bash
helm repo add grafana-community https://grafana-community.github.io/helm-charts
helm repo update grafana-community

kubectl --context=sre -n monitoring create configmap grafana-dashboards \
  --from-file=k8s/grafana/dashboards \
  --dry-run=client -o yaml | kubectl --context=sre apply -f -

helm upgrade --install grafana grafana-community/grafana \
  --version 14.4.1 \
  --kube-context sre \
  --namespace monitoring --create-namespace \
  --values k8s/grafana/values.yaml \
  --wait --timeout 5m

kubectl --context=sre -n monitoring get pods,services,pvc -l app.kubernetes.io/instance=grafana
```

Одна реплика, PVC 1 GiB для настроек и будущих дашбордов. При обновлении сначала останавливается прежний Pod, затем запускается новый. Контейнер работает от непривилегированного пользователя чарта; доступ к Kubernetes API ему не нужен.

Namespace `monitoring` уже должен существовать после установки VictoriaMetrics. ConfigMap содержит JSON из папки `dashboards`; Grafana читает их с диска, без скачивания из интернета и дополнительных sidecar-контейнеров.

## Вход

В отдельном терминале:

```bash
kubectl --context=sre -n monitoring port-forward service/grafana 3000:80
```

Открыть [localhost:3000](http://localhost:3000/). Логин — `admin`. Получить пароль:

```bash
kubectl --context=sre -n monitoring get secret grafana \
  -o jsonpath='{.data.admin-password}' | base64 --decode
echo
```

## Проверка источника

В Grafana открыть **Connections → Data sources → VictoriaMetrics** и нажать **Save & test**. Затем в **Explore** выбрать VictoriaMetrics и выполнить:

```promql
up{namespace="wishlist"}
```

Должны появиться метрики работающих сборов. Пустой ответ при успешной проверке подключения означает, что нужно проверить vmagent и его targets.

Grafana обращается к `http://victoria-metrics:8428` внутри namespace `monitoring`. Пробрасывать VictoriaMetrics на localhost для этого не нужно. `localhost` внутри Grafana означал бы сам контейнер Grafana.

## Дашборды

В **Dashboards → SRE** появятся два дашборда:

- [Wishlist · приложение и SLA](http://localhost:3000/d/wishlist-service) — JSON в `dashboards/wishlist.json`.
- [PostgreSQL Exporter · Wishlist](http://localhost:3000/d/wishlist-postgres) — JSON в `dashboards/postgres.json`.

### Wishlist

| Строка | Что показывает |
|---|---|
| Обзор | Проверка пользовательского сценария, RPS, доля `5xx`, p95 задержки |
| HTTP | RPS по статусам и маршрутам, p50/p95/p99 |
| Go runtime | Горутины, heap и RSS, CPU процесса, доступность `/metrics` |
| Бизнес-операции | Успешные операции и отказы по причинам за 5 минут |
| Пробер | Последний результат, его возраст, длительность проверки и доступность относительно SLA |

Переменная **Pod (runtime)** меняет только runtime-панели. Остальные показатели считаются по всем репликам. HTTP-панели исключают frontend и health checks. Counter сначала обрабатывается через `rate` или `increase`, затем результаты складываются по Pods. При отсутствии трафика доля ошибок и квантили задержки не определены.

CPU показан в процентах одного ядра: 100% — одно ядро, а не лимит контейнера. Heap и RSS различаются: heap отражает выделенные Go-объекты, RSS — резидентную память всего процесса. `up` показывает сбор метрик и не заменяет пользовательскую проверку.

### Пробер и SLA

Для SLA используется [прежняя формула](../slo/queries.md): `100 × хорошие минуты / 10 ожидаемых минут`, с ограничением результата до 100%. Цель — 99,5%. Gauge показывает текущее значение, график — изменение и постоянную линию цели. Пропущенные минуты не исключаются из знаменателя.

После запуска дать проберу минимум 11 минут. Пока окно не заполнено, значение будет ниже 100%. Это демонстрационное окно, не подтверждение недельного SLA. Учёт через `increase` приближённый из-за дискретного сбора метрик.

Последний результат показывается только при успешном сборе метрик и завершённой проверке не старше минуты. При отсутствии свежих наблюдений панель показывает **нет данных**, а не успешную проверку. Возраст результата помогает заметить зависший пробер.

### PostgreSQL

Импортирован [PostgreSQL Database, ID 9628](https://grafana.com/grafana/dashboards/9628-postgresql-database/), **revision 8**, автор — `lstn/misc-grafana-dashboards`. Локальная копия адаптирована под нашу конфигурацию:

- Источник заменён на VictoriaMetrics, переменные — на наши labels `namespace`, `app`, `instance`, `datname` и `mode`.
- Старые панели `graph` и `singlestat` заменены на Time series и Stat.
- CPU, RSS и файловые дескрипторы подписаны как показатели **exporter**, а не PostgreSQL. Для RSS исправлен исходный запрос: берётся объём памяти, а не скорость его изменения.
- Вместо времени старта показан `pg_up`: метрика старта не включена в стандартный набор нашего exporter. Удалены старые checkpoint/backend-метрики bgwriter, отсутствующие в PostgreSQL 18. Дополнительные collectors ради них не включались.

Оставлены настройки PostgreSQL, сессии, транзакции, строки, блокировки, cache hit rate, bgwriter, конфликты, deadlocks и временные файлы. Нулевые значения не скрываются. Процент попаданий в кеш может отсутствовать, если ещё не было обращений к блокам.

Выбрать **Exporter** и **База данных**. На стенде проверить `pg_up`, сессии и транзакции: после действий в Wishlist счётчики БД должны меняться. Ссылка вверху каждого дашборда ведёт на второй с сохранением периода.

## Изменения и экспорт

После изменения JSON повторить команду создания ConfigMap из раздела установки. Kubernetes обновит файлы, затем Grafana перечитает их; это может занять несколько минут. Изменения в UI разрешены, но при обновлении файлов Grafana перезапишет их. Чтобы сохранить правки, экспортировать JSON обратно в проект.

JSON дашбордов находятся в `dashboards/` в формате Classic. Эти файлы используются для установки и входят в архив с отчётом, скриншотами и записью. При экспорте изменений из Grafana выбирать формат Classic. Дашборды можно также импортировать вручную через **Dashboards → New → Import**; источник имеет UID `victoria-metrics`.

Если Grafana не запускается:

```bash
kubectl --context=sre -n monitoring get pods,pvc
kubectl --context=sre -n monitoring logs deployment/grafana --tail=50
kubectl --context=sre -n monitoring get events --sort-by=.metadata.creationTimestamp
```

Документация: [Helm-чарт](https://github.com/grafana-community/helm-charts/tree/main/charts/grafana), [Prometheus data source](https://grafana.com/docs/grafana/latest/datasources/prometheus/configure/).
