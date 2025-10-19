# REST-сервис для агрегации данных об онлайн-подписках пользователей.
env-переменные для локального запуска описаны в одном файле.
Примеры файлов находятся в директории `.env` и имеют расширение `.example`.
Перед запуском достаточно подготовить единый файл окружения для docker-compose:

```
cp .env/local.env.example .env/local.env
```

Внутри файла можно указать все необходимые переменные, например:

```env
APP_ENV=local
HTTP_HOST=0.0.0.0
HTTP_PORT=8080
HTTP_TIMEOUT=5s
HTTP_CORS_ORIGINS=http://localhost:8082,http://127.0.0.1:8082

POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=subs_user
POSTGRES_PASSWORD=subs_password
POSTGRES_DB=subs_db
POSTGRES_SSLMODE=disable

PG_PORT_HOST=5433
PG_PORT_CONTAINER=5432

ADMINER_PORT_HOST=8081
ADMINER_PORT_CONTAINER=8080

APP_PORT_HOST=8080
APP_PORT_CONTAINER=8080

SWAGGER_PORT_HOST=8082
SWAGGER_PORT_CONTAINER=8080
```

Запуск через контейнер:
1. Указать желаемый путь к `ENV_FILE` в docker-compose.yml (по умолчанию используется `.env/local.env`).
2. Запустить run_service в Makefile.
3. Подождать полного запуска.
4. Запустить migrate-up в Makefile.
5. На localhost:8082 с дефолтными параметрами Swagger UI для ручного тестирования, с примерами запросов.

