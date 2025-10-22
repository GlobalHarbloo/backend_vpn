#!/bin/bash

echo "🔧 Fixing import paths..."

# Находим все Go файлы и заменяем импорты
find . -name "*.go" -type f -exec sed -i 's|vpn-backend/|github.com/yourusername/vpn-backend/|g' {} \;

echo "✅ Import paths fixed!"
echo "Now you can run: go build -o vpn-backend ./cmd/server"
