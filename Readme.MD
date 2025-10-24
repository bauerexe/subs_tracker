# Subs Tracker

REST-сервис для управления подписками пользователей.

## Быстрый запуск

1. Скопируйте шаблон окружения и при необходимости измените значения:
   ```bash
   cp .env/local.env.example .env/local.env
   ```
2. Запустите весь сервис (миграции будут применены автоматически):
   ```bash
   make run_service
   ```

## Makefile

|                | Назначение                                                                   |
|----------------|------------------------------------------------------------------------------|
| `run_service`  | Собрать и запустить приложение вместе с БД + миграции, Adminer и Swagger UI. |
| `up`           | Поднять только PostgreSQL и Adminer в фоне.                                  |
| `down`         | Остановить все контейнеры.                                                   |
| `down-v`       | Остановить контейнеры и удалить тома.                                        |
| `restart`      | Перезапустить инфраструктуру (`down` + `up`).                                |
| `migrate-up`   | Применить все миграции.                                                      |
| `migrate-down` | Откатить последнюю миграцию.                                                 |

## Файл окружения `.env/local.env`

| Переменная               | Описание                                                                                |
|--------------------------|-----------------------------------------------------------------------------------------|
| `APP_ENV`                | Текущий профиль запуска сервиса.                                                        |
| `HTTP_HOST`              | Адрес интерфейса, на котором слушает HTTP-сервер.                                       |
| `HTTP_PORT`              | Порт HTTP-сервера внутри контейнера.                                                    |
| `HTTP_TIMEOUT`           | Таймаут обработки HTTP-запроса.                                                         |
| `HTTP_CORS_ORIGINS`      | Список доменов, которым разрешены CORS-запросы.                                         |
| `POSTGRES_HOST`          | Хост PostgreSQL из контейнера приложения.                                               |
| `POSTGRES_PORT`          | Порт PostgreSQL из контейнера приложения.                                               |
| `POSTGRES_USER`          | Пользователь базы данных.                                                               |
| `POSTGRES_PASSWORD`      | Пароль пользователя базы данных.                                                        |
| `POSTGRES_DB`            | Имя базы данных.                                                                        |
| `POSTGRES_SSLMODE`       | Режим SSL для подключения к PostgreSQL.                                                 |
| `PG_PORT_HOST`           | Порт PostgreSQL, проброшенный на хост (подключение `psql` к `localhost:$PG_PORT_HOST`). |
| `PG_PORT_CONTAINER`      | Внутренний порт PostgreSQL внутри docker-compose.                                       |
| `ADMINER_PORT_HOST`      | Порт Adminer на хосте (`http://localhost:$ADMINER_PORT_HOST`).                          |
| `ADMINER_PORT_CONTAINER` | Внутренний порт Adminer.                                                                |
| `APP_PORT_HOST`          | Порт приложения, проброшенный на хост.                                                  |
| `APP_PORT_CONTAINER`     | Внутренний порт приложения внутри docker-compose.                                       |
| `SWAGGER_PORT_HOST`      | Порт Swagger UI на хосте (`http://localhost:$SWAGGER_PORT_HOST`).                       |
| `SWAGGER_PORT_CONTAINER` | Внутренний порт Swagger UI.                                                             |

## Приоритет конфигурации
1. `docker-compose.yml` задаёт базовые значения для сервисов и при необходимости читает переопределения из `.env` или файла, переданного в `docker compose --env-file`.
2. Базовые значения зашиты в коде (`internal/config/config.go`) и совпадают с настройками из `docker-compose.yml`, поэтому сервис поднимается даже без внешнего файла.
3. Если выставлена переменная `ENV_FILE` (по умолчанию `local.env`) и указанный файл существует, значения загружаются из него и перекрывают базовые настройки.
4. Если файл не найден, конфигурация ищет переменные окружения процесса и применяет их поверх базовых значений.

## URL

- Приложение: `http://localhost:${APP_PORT_HOST}`
- Swagger UI: `http://localhost:${SWAGGER_PORT_HOST}`
- Adminer: `http://localhost:${ADMINER_PORT_HOST}` (сервер `postgres`, пользователь `POSTGRES_USER`, пароль
  `POSTGRES_PASSWORD`)

## Кодогенерация

| Команда         | Где                                      |
|-----------------|------------------------------------------|
| `go:generate`   | `internal/usecase/usecase.go`            |
| `sqlc generate` | терминал                                 | 
| `go:generate`   | `internal/entity/generated/generated.go` |