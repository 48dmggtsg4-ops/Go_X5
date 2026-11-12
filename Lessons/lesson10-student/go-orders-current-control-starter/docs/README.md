# Swagger-документация

Документация генерируется из комментариев в исходном коде.

Из корня Go-модуля:

```bash
go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g cmd/api/main.go -o docs --parseInternal
```

После запуска сервера Swagger UI доступен по адресу:

http://localhost:8080/swagger/index.html

Файлы docs.go, swagger.json и swagger.yaml сохраняются в Git.