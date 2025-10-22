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

# Удаляем все sagernet зависимости из go.mod
echo "Removing problematic sagernet dependencies..."
sed -i '/sagernet/d' go.mod

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
    echo "Trying alternative approach..."
    
    # Альтернативный подход - полная очистка
    echo "Performing full cleanup..."
    go clean -modcache
    rm -f go.sum
    go mod download
    go mod tidy
    go build -o vpn-backend ./cmd/server
    
    if [ $? -eq 0 ]; then
        echo "✅ Build successful after cleanup!"
    else
        echo "❌ Build still failed!"
        exit 1
    fi
fi
