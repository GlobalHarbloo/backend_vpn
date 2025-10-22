# 🚀 Deployment Guide

## Установка и запуск на сервере

### 1. Подготовка сервера

```bash
# Обновляем систему
sudo apt update && sudo apt upgrade -y

# Устанавливаем Go (если не установлен)
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Устанавливаем PostgreSQL
sudo apt install postgresql postgresql-contrib -y
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Создаем базу данных
sudo -u postgres psql
CREATE DATABASE vpn;
CREATE USER vpnuser WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE vpn TO vpnuser;
\q
```

### 2. Клонирование и настройка проекта

```bash
# Клонируем репозиторий
git clone https://github.com/GlobalHarbloo/backend_vpn.git
cd backend_vpn

# Устанавливаем зависимости
go mod tidy
go mod download

# Собираем проект
go build -o vpn-backend ./cmd/server
```

### 3. Настройка переменных окружения

```bash
# Создаем файл с переменными окружения
cat > .env << EOF
# База данных
DATABASE_URL=postgres://vpnuser:your_password@localhost:5432/vpn?sslmode=disable

# Сервер
SERVER_PORT=8081
JWT_SECRET=your-super-secret-jwt-key-here
ADMIN_TOKEN=your-admin-token-here

# Xray
XRAY_CONFIG_PATH=/etc/xray/config.json
XRAY_TEMPLATE_PATH=/etc/xray/config_template.json

# Email
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASS=your-app-password

# Фронтенд
FRONTEND_URL=https://your-domain.com

# ЮKassa (опционально)
YOOKASSA_SHOP_ID=your-shop-id
YOOKASSA_SECRET=your-secret
YOOKASSA_RETURN_URL=https://your-domain.com/payment-success
EOF

# Загружаем переменные
export $(cat .env | xargs)
```

### 4. Настройка Xray

```bash
# Устанавливаем Xray
bash <(curl -L https://raw.githubusercontent.com/XTLS/Xray-install/main/install-release.sh)

# Создаем директорию для конфигов
sudo mkdir -p /etc/xray

# Создаем базовый конфиг Xray
sudo tee /etc/xray/config.json > /dev/null << 'EOF'
{
  "api": {
    "services": ["StatsService"],
    "tag": "api"
  },
  "inbounds": [
    {
      "listen": "127.0.0.1",
      "port": 10085,
      "protocol": "dokodemo-door",
      "settings": {
        "address": "127.0.0.1"
      },
      "tag": "api",
      "network": "tcp"
    },
    {
      "listen": "0.0.0.0",
      "port": 1443,
      "protocol": "vless",
      "settings": {
        "clients": [],
        "decryption": "none"
      },
      "streamSettings": {
        "network": "ws",
        "security": "none",
        "wsSettings": {
          "path": "/openai"
        }
      }
    }
  ],
  "outbounds": [
    {
      "protocol": "freedom",
      "settings": {
        "domainStrategy": "UseIP"
      },
      "tag": "direct"
    }
  ],
  "routing": {
    "domainStrategy": "IPIfNonMatch",
    "rules": [
      {
        "inboundTag": ["api"],
        "outboundTag": "api",
        "type": "field"
      }
    ]
  },
  "stats": {},
  "policy": {
    "levels": {
      "0": {
        "statsUserDownlink": true,
        "statsUserUplink": true
      }
    },
    "system": {
      "statsInboundDownlink": true,
      "statsInboundUplink": true
    }
  }
}
EOF

# Создаем template для Xray
sudo tee /etc/xray/config_template.json > /dev/null << 'EOF'
{
  "api": {
    "services": ["StatsService"],
    "tag": "api"
  },
  "inbounds": [
    {
      "listen": "127.0.0.1",
      "port": 10085,
      "protocol": "dokodemo-door",
      "settings": {
        "address": "127.0.0.1"
      },
      "tag": "api",
      "network": "tcp"
    },
    {
      "listen": "0.0.0.0",
      "port": 1443,
      "protocol": "vless",
      "settings": {
        "clients": [
          {{range $i, $user := .Users}}
          {{if $i}},{{end}}
          {
            "id": "{{$user.UUID}}",
            "email": "{{$user.Email}}",
            "level": 0,
            "alterId": 0
          }
          {{end}}
        ],
        "decryption": "none"
      },
      "streamSettings": {
        "network": "ws",
        "security": "none",
        "wsSettings": {
          "path": "/openai"
        }
      }
    }
  ],
  "outbounds": [
    {
      "protocol": "freedom",
      "settings": {
        "domainStrategy": "UseIP"
      },
      "tag": "direct"
    }
  ],
  "routing": {
    "domainStrategy": "IPIfNonMatch",
    "rules": [
      {
        "inboundTag": ["api"],
        "outboundTag": "api",
        "type": "field"
      }
    ]
  },
  "stats": {},
  "policy": {
    "levels": {
      "0": {
        "statsUserDownlink": true,
        "statsUserUplink": true
      }
    },
    "system": {
      "statsInboundDownlink": true,
      "statsInboundUplink": true
    }
  }
}
EOF

# Запускаем Xray
sudo systemctl start xray
sudo systemctl enable xray
```

### 5. Создание systemd сервиса

```bash
# Создаем systemd сервис
sudo tee /etc/systemd/system/vpn-backend.service > /dev/null << EOF
[Unit]
Description=VPN Backend Service
After=network.target postgresql.service

[Service]
Type=simple
User=root
WorkingDirectory=/root/backend_vpn
ExecStart=/root/backend_vpn/vpn-backend
Restart=always
RestartSec=5
EnvironmentFile=/root/backend_vpn/.env

[Install]
WantedBy=multi-user.target
EOF

# Перезагружаем systemd и запускаем сервис
sudo systemctl daemon-reload
sudo systemctl enable vpn-backend
sudo systemctl start vpn-backend

# Проверяем статус
sudo systemctl status vpn-backend
```

### 6. Настройка Nginx (опционально)

```bash
# Устанавливаем Nginx
sudo apt install nginx -y

# Создаем конфиг для прокси
sudo tee /etc/nginx/sites-available/vpn-backend > /dev/null << 'EOF'
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8081;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
EOF

# Активируем конфиг
sudo ln -s /etc/nginx/sites-available/vpn-backend /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

### 7. Проверка работы

```bash
# Проверяем логи
sudo journalctl -u vpn-backend -f

# Тестируем API
curl -X POST http://localhost:8081/register \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"password123"}'

# Проверяем статус Xray
sudo systemctl status xray
```

## 🔧 Troubleshooting

### Если go mod tidy не работает:

```bash
# Очищаем кэш модулей
go clean -modcache

# Удаляем go.sum и пересоздаем
rm go.sum
go mod tidy
```

### Если есть проблемы с зависимостями:

```bash
# Принудительно скачиваем все зависимости
go mod download -x
go mod verify
```

### Если Xray не запускается:

```bash
# Проверяем конфиг
sudo xray -test -config /etc/xray/config.json

# Проверяем логи
sudo journalctl -u xray -f
```
