# REST-сервис для агрегации данных об онлайн-подписках пользователей.
env-переменные для локального запуска:
Примеры файлов находятся в директории `.env` и имеют расширение `.example`.
Перед запуском достаточно подготовить переменные окружения для docker-compose:

```
cp .env/local_pg.env.example .env/local_pg.env
```

1. CONFIG_PATH - путь до файла с конфигом
Пример: configs/local.yaml (файл уже находится в репозитории, путь можно переопределить через `CONFIG_PATH`)
```yaml
env: "local"
http_server:
  host: "..."
  port: ...
  timeout: 4s
postgres:
  host: "..."
  port: ...
  user: ${POSTGRES_USER}
  password: ${POSTGRES_PASSWORD}
  db: ${POSTGRES_DB}
  sslmode: "..."
```
2. CONFIG_PG_PATH - путь до файла с конфигом для postgres:
Пример: .env
```env
POSTGRES_USER=...
POSTGRES_PASSWORD=...
POSTGRES_DB=...

PG_PORT_HOST=...
PG_PORT_CONTAINER=...

ADMINER_PORT_HOST=...
ADMINER_PORT_CONTAINER=...
```

Запуск через контейнер:
1. указать желаемые пути для       CONFIG_PATH CONFIG_PG_PATH - в docker-compose.yml
2. запусть run_service в Makefile
3. подождать полного запуска 
4. запустить migrate-up в Makefile
5. нa localhost:8082 c дефолтными параметрами Swagger UI для ручного тестирования, с примерами запросов

