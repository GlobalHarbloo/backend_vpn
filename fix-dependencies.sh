#!/bin/bash

echo "🔧 Fixing Go dependencies..."

# Очищаем кэш модулей
echo "Cleaning module cache..."
go clean -modcache

# Удаляем go.sum если есть проблемы
if [ -f "go.sum" ]; then
    echo "Removing go.sum..."
    rm go.sum
fi

# Скачиваем все зависимости принудительно
echo "Downloading dependencies..."
go mod download -x

# Проверяем зависимости
echo "Verifying dependencies..."
go mod verify

# Обновляем зависимости
echo "Tidying modules..."
go mod tidy

# Проверяем сборку
echo "Testing build..."
go build -o vpn-backend ./cmd/server

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo "Executable created: ./vpn-backend"
else
    echo "❌ Build failed!"
    exit 1
fi
