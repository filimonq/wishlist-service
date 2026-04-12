# wishlist-service

REST API сервис для управления вишлистами подарков. 
Пользователь может зарегистрироваться, создать вишлист к событию, добавить туда желаемые подарки 
и поделиться публичной ссылкой — без регистрации люди смогут посмотреть список и забронировать подарок.

## Стек

- **Go 1.25** — gin, pgx, golang-jwt, golang-migrate
- **PostgreSQL 18**
- **Docker Compose**

## Запуск

```bash
make up
```

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
curl http://localhost:8080/api/v1/wishlists/<id> \
  -H "Authorization: Bearer <token>"

# обновить вишлист
curl -X PUT http://localhost:8080/api/v1/wishlists/<id> \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"title": "Новое название"}'

# удалить вишлист
curl -X DELETE http://localhost:8080/api/v1/wishlists/<id> \
  -H "Authorization: Bearer <token>"
```

- Подарки

```bash
# добавить подарок
curl -X POST http://localhost:8080/api/v1/wishlists/<id>/items \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"name": "PlayStation 5", "description": "ну прям очень хочу", "url": "https://example.com", "priority": 5}'

# список подарков
curl http://localhost:8080/api/v1/wishlists/<id>/items \
  -H "Authorization: Bearer <token>"

# обновить подарок
curl -X PUT http://localhost:8080/api/v1/wishlists/<id>/items/<itemId> \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"priority": 3}'

# удалить подарок
curl -X DELETE http://localhost:8080/api/v1/wishlists/<id>/items/<itemId> \
  -H "Authorization: Bearer <token>"
```

- Публичный доступ

```bash
# получить вишлист по токену (без авторизации)
curl http://localhost:8080/api/v1/public/<public_token>

# забронировать подарок (без авторизации)
curl -X POST http://localhost:8080/api/v1/public/<public_token>/items/<itemId>/reserve
```




## Нагрузочное тестирование
Скрипт лежит в корне проекта (k6.js)
Тест проверяет атомарность бронирования — 50 VU одновременно ломятся на один подарок.

```bash
make k6
```

Результаты на локальной машине (0.5 CPU, 128MB):

|         Метрика           |      Значение     |
|---------------------------|-------------------|
| Победителей (reserve 204) | 1 из 7350 попыток |
| Конфликтов (reserve 409)  | 7349              |
| Ошибок                    | 0                 |
| Латентность p95           | 3.12ms            |
| Throughput                | ~480 rps          |