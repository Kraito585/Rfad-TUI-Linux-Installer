#!/usr/bin/env bash

# Останавливать выполнение при любой ошибке (например, если код не скомпилировался)
set -e 

# Функция для сборки, чтобы не писать одно и то же дважды
build_and_hash() {
    local TAG=$1
    local OUTPUT_NAME="rfad-installer-$TAG"

    echo "==> Собираем версию: $TAG"
    
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags "$TAG" -ldflags="-s -w" -o "$OUTPUT_NAME" .

    # Высчитываем хэш
    local HASH=$(sha256sum "$OUTPUT_NAME" | awk '{print $1}')

    # Переименовываем
    local NEW_NAME="$OUTPUT_NAME-$HASH"
    mv "$OUTPUT_NAME" "$NEW_NAME"

    echo "==> Успешно: $NEW_NAME"
    echo "------------------------------------------------"
}

rm -f rfad-installer-*

build_and_hash "gdrive"
build_and_hash "cdn"

echo "Все сборки завершены!"