# wishlist-service

Веб-приложение для управления вишлистами подарков.
Пользователь может зарегистрироваться, создать вишлист к событию, добавить туда желаемые подарки 
и поделиться публичной ссылкой — без регистрации люди смогут посмотреть список и забронировать подарок.

## Стек

- **Go 1.25** — gin, pgx, golang-jwt, golang-migrate
- **PostgreSQL 18**
- **HTML, CSS, vanilla JavaScript** — пользовательский интерфейс
- **Docker Compose**

Frontend и REST API доступны через один Go-сервер. Файлы интерфейса встроены в бинарник, отдельная сборка frontend не требуется.

## Архитектура
Проект построен по принципам гексагональной архитектуры (Ports & Adapters):

```
cmd/
  wishlist/                 — запуск HTTP-сервера
  migrate/                  — отдельная команда миграций
frontend/                   — HTML, CSS, JavaScript и встраивание файлов в бинарник
internal/
  app/                      — конфигурация, DI, запуск сервера
  domain/                   — бизнес-сущности и ошибки
  service/                  — бизнес-логика, ports
  adapter/
    in/http/                — HTTP-хендлеры, роутер, middleware, DTO
    out/repository/         — реализации репозиториев (PostgreSQL, pgx)
  migrate/                  — применение SQL-миграций
migrations/                 — SQL-миграции
```

## Запуск

Для запуска нужны Docker с Docker Compose и make. Команды выполняются из корня проекта.

Создайте `.env`, если его ещё нет:

```bash
test -f .env || cp .env.example .env
```

Заполните `JWT_SECRET` в `.env`. Случайное значение можно получить командой:

```bash
openssl rand -hex 32
```

Соберите образ, сохраните конфигурацию релиза и запустите приложение:

```bash
make build VERSION=v1
make release VERSION=v1
make up VERSION=v1
```

Compose поднимет PostgreSQL, выполнит миграции отдельным контейнером и запустит backend. Интерфейс доступен по адресу [http://localhost:8080](http://localhost:8080).

Зарегистрируйтесь, создайте вишлист с будущей датой события и добавьте подарки. Публичную ссылку можно отправить друзьям: просмотр и бронирование доступны без регистрации.

### Конфигурация

| Переменная | Назначение |
|---|---|
| `HTTP_PORT` | Порт Go-сервера внутри контейнера, по умолчанию `8080` |
| `HOST_PORT` | Опубликованный порт на хосте, по умолчанию `8080` |
| `DB_CONN` | Строка подключения backend и миграций к PostgreSQL; обязательна |
| `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD` | Настройки создаваемой PostgreSQL; должны соответствовать `DB_CONN` |
| `JWT_SECRET` | Секрет подписи JWT; обязателен, одинаковый для всех экземпляров backend |
| `MIGRATIONS_PATH` | Путь к SQL-миграциям, по умолчанию `migrations` |

Пример `DB_CONN` рассчитан на запуск в Compose: `postgres` — имя сервиса БД внутри Docker-сети. `.env` и `.env.VERSION` содержат локальную конфигурацию и исключены из Git.

### Управление запуском

```bash
make logs VERSION=v1       # логи backend
make migrate VERSION=v1    # повторное применение недостающих миграций
make down VERSION=v1       # остановка и удаление контейнеров, данные БД сохраняются
```

`make release` сохраняет `.env` в `.env.VERSION`. После изменения кода или конфигурации используйте новую версию, например `v2`, и повторите build, release и up. Команды не перезаписывают существующие версии образа и конфигурации.

### Несколько экземпляров backend

Перед созданием нового релиза задайте в `.env` значение `HOST_PORT=0`, чтобы Docker назначил каждому экземпляру отдельный порт:

```bash
make build VERSION=v2
make release VERSION=v2
make scale VERSION=v2 REPLICAS=2
```

Команда покажет контейнеры и назначенные порты. Все экземпляры используют общую PostgreSQL и JWT-секрет. Балансировщик в Compose не включён.

## Swagger

После запуска документация доступна по адресу:
```
http://localhost:8080/swagger/index.html
```


## API
- Авторизация

```bash
# регистрация
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'

# логин
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123"}'
```

- Вишлисты

```bash
# создать вишлист
curl -X POST http://localhost:8080/api/v1/wishlists \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title": "День рождения", "description": "Мой вишлист", "event_date": "2027-09-01T00:00:00Z"}'

# список вишлистов
curl http://localhost:8080/api/v1/wishlists \
  -H "Authorization: Bearer <token>"

# получить вишлист
curl 'http://localhost:8080/api/v1/wishlists/<id>' \
  -H "Authorization: Bearer <token>"

# обновить вишлист
curl -X PUT 'http://localhost:8080/api/v1/wishlists/<id>' \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title": "Новое название"}'

# удалить вишлист
curl -X DELETE 'http://localhost:8080/api/v1/wishlists/<id>' \
  -H "Authorization: Bearer <token>"
```

- Подарки

```bash
# добавить подарок
curl -X POST 'http://localhost:8080/api/v1/wishlists/<id>/items' \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "PlayStation 5", "description": "ну прям очень хочу", "url": "https://example.com", "priority": 5}'

# список подарков
curl 'http://localhost:8080/api/v1/wishlists/<id>/items' \
  -H "Authorization: Bearer <token>"

# обновить подарок
curl -X PUT 'http://localhost:8080/api/v1/wishlists/<id>/items/<itemId>' \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"priority": 3}'

# удалить подарок
curl -X DELETE 'http://localhost:8080/api/v1/wishlists/<id>/items/<itemId>' \
  -H "Authorization: Bearer <token>"
```

- Публичный доступ

```bash
# получить вишлист по токену (без авторизации)
curl 'http://localhost:8080/api/v1/public/<public_token>'

# забронировать подарок (без авторизации)
curl -X POST 'http://localhost:8080/api/v1/public/<public_token>/items/<itemId>/reserve'
```

## Тестирование

Для тестов и линтера нужен Go 1.25 или новее. Версии инструментов закреплены в Makefile; при первом запуске потребуется скачивание зависимостей. `make test` автоматически генерирует моки через mockery.

```bash
# генерация моков
make mocks

# запуск всех тестов
make test

# запуск линтера
make lint

# проверка с детектором гонок после генерации моков
go test -race ./...

# обновление Swagger после изменения аннотаций API
make swag
```

## Нагрузочное тестирование
Скрипт `k6/race_cond_test.js` проверяет атомарность бронирования: 50 виртуальных пользователей одновременно пытаются забронировать один подарок. k6 запускается в Docker. Backend должен быть доступен по указанному адресу.

```bash
make k6

# если приложение доступно на другом порту
BASE_URL=http://localhost:9000/api/v1 make k6
```

По умолчанию используется `http://localhost:8080/api/v1`. Команда использует Docker-сеть `host`, рассчитанную на Linux.

Успешное бронирование должно быть ровно одно, остальные попытки получают `409 Conflict`. k6 учитывает эти ожидаемые ответы в `http_req_failed`; для результата теста используются счётчики `reserve_success` и `reserve_error`.

Пример локального прогона с лимитами backend 0.5 CPU и 128 МБ:

|         Метрика           |      Значение     |
|---------------------------|-------------------|
| Победителей (reserve 204) | 1 из 7358 попыток |
| Конфликтов (reserve 409)  | 7357              |
| Ошибок                    | 0                 |
| Латентность p95           | 2.69ms            |
| Throughput                | ~481 rps          |
