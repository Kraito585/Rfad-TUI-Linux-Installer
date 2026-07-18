#!/usr/bin/env bash

# Останавливать выполнение при любой ошибке
set -e 

build_and_hash() {
    local TAG=$1
    local OUTPUT_NAME="rfad-installer-$TAG"

    echo "==> Собираем версию: $TAG"
    
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "$TAG" -ldflags="-s -w" -o "$OUTPUT_NAME" .

    # Высчитываем хэш
    local HASH=$(sha256sum "$OUTPUT_NAME" | awk '{print $1}')

    # Имя для файла-маркера
    local HASH_FILENAME="sha256-${OUTPUT_NAME}-${HASH}"

    # Создаем пустой файл с хэшем в названии
    touch "$HASH_FILENAME"

    echo "==> Успешно собран: $OUTPUT_NAME"
    echo "==> Создан маркер: $HASH_FILENAME"
    echo "------------------------------------------------"
}

# Очистка старых билдов и старых файлов с хэшами
rm -f rfad-installer-*
rm -f sha256-*

# Запускаем сборки
build_and_hash "gdrive"
build_and_hash "cdn"

echo "Все сборки завершены!"