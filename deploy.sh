#!/bin/bash

set -e

# Цвета для вывода
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 DevOps Manual Deployment Script${NC}"

# Проверка запуска от root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}❌ Пожалуйста, запустите скрипт с sudo${NC}"
    exit 1
fi

# Переменные
PROJECT_NAME="devops-manual"
PROJECT_DIR="/var/www/${PROJECT_NAME}"
SERVICE_NAME="${PROJECT_NAME}"
DB_NAME="devops_manual"
DB_USER="devops_user"
GO_VERSION="1.21.6"

# Получаем данные от пользователя
read -p "Введите домен (например: mvp2001.ru): " DOMAIN
read -p "Введите пароль для PostgreSQL: " DB_PASSWORD
read -p "Введите токен Telegram бота: " TELEGRAM_TOKEN
read -p "Введите ID чата Telegram: " TELEGRAM_CHAT_ID
read -p "Введите пароль для админа: " ADMIN_PASSWORD

echo -e "${YELLOW}📦 Установка зависимостей...${NC}"
apt-get update
apt-get install -y postgresql postgresql-contrib nginx git curl wget certbot python3-certbot-nginx

# Установка Go если не установлен
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}🔧 Установка Go...${NC}"
    wget "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
    rm "go${GO_VERSION}.linux-amd64.tar.gz"
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    export PATH=$PATH:/usr/local/go/bin
fi

echo -e "${YELLOW}🐘 Настройка PostgreSQL...${NC}"
sudo -u postgres psql << PSQL
CREATE USER ${DB_USER} WITH PASSWORD '${DB_PASSWORD}';
CREATE DATABASE ${DB_NAME} OWNER ${DB_USER};
GRANT ALL PRIVILEGES ON DATABASE ${DB_NAME} TO ${DB_USER};
PSQL

echo -e "${YELLOW}📁 Создание директорий...${NC}"
mkdir -p ${PROJECT_DIR}
mkdir -p /var/log/${PROJECT_NAME}

# Клонирование репозитория
if [ -d "${PROJECT_DIR}/.git" ]; then
    echo -e "${YELLOW}🔄 Обновление репозитория...${NC}"
    cd ${PROJECT_DIR} && git pull
else
    echo -e "${YELLOW}📥 Клонирование репозитория...${NC}"
    read -p "Введите URL GitHub репозитория: " REPO_URL
    git clone ${REPO_URL} ${PROJECT_DIR}
fi

echo -e "${YELLOW}⚙️ Создание .env файла...${NC}"
cat > ${PROJECT_DIR}/.env << ENV
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=${DB_USER}
DB_PASSWORD=${DB_PASSWORD}
DB_NAME=${DB_NAME}

# Telegram Bot
TELEGRAM_BOT_TOKEN=${TELEGRAM_TOKEN}
TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID}

# Server
SERVER_PORT=8080
DOMAIN=${DOMAIN}

# Admin
ADMIN_PASSWORD=${ADMIN_PASSWORD}
ENV

echo -e "${YELLOW}🔨 Сборка приложения...${NC}"
cd ${PROJECT_DIR}
export PATH=$PATH:/usr/local/go/bin
go mod tidy
go build -o ${PROJECT_NAME} cmd/main.go

echo -e "${YELLOW}👤 Создание администратора...${NC}"
cat > /tmp/create_admin.go << 'GOADMIN'
package main
import (
    "database/sql"
    "fmt"
    "os"
    "github.com/joho/godotenv"
    _ "github.com/lib/pq"
    "golang.org/x/crypto/bcrypt"
)
func main() {
    godotenv.Load()
    connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
        os.Getenv("DB_HOST"), os.Getenv("DB_PORT"), os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"), os.Getenv("DB_NAME"))
    db, err := sql.Open("postgres", connStr)
    if err != nil { panic(err) }
    defer db.Close()
    
    password := os.Getenv("ADMIN_PASSWORD")
    hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    
    _, err = db.Exec("INSERT INTO users (username, password_hash, is_admin) VALUES ('admin', $1, true) ON CONFLICT (username) DO UPDATE SET password_hash = $1", string(hash))
    if err != nil { panic(err) }
    fmt.Println("✅ Admin создан")
}
GOADMIN

cd /tmp && go mod init tmpadmin && go get github.com/joho/godotenv github.com/lib/pq golang.org/x/crypto/bcrypt
cp ${PROJECT_DIR}/.env /tmp/.env
go run create_admin.go

echo -e "${YELLOW}🔧 Настройка Nginx...${NC}"
cat > /etc/nginx/sites-available/${PROJECT_NAME} << NGINX
server {
    listen 80;
    server_name ${DOMAIN} www.${DOMAIN};
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_cache_bypass \$http_upgrade;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }

    location /static/ {
        alias ${PROJECT_DIR}/web/static/;
        expires 1y;
        add_header Cache-Control "public, immutable";
    }
}
NGINX

ln -sf /etc/nginx/sites-available/${PROJECT_NAME} /etc/nginx/sites-enabled/
rm -f /etc/nginx/sites-enabled/default
nginx -t && systemctl reload nginx

echo -e "${YELLOW}🔒 Получение SSL сертификата...${NC}"
certbot --nginx -d ${DOMAIN} --non-interactive --agree-tos -m admin@${DOMAIN} || true

echo -e "${YELLOW}⚙️ Создание systemd сервиса...${NC}"
cat > /etc/systemd/system/${SERVICE_NAME}.service << SERVICE
[Unit]
Description=DevOps Manual
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
WorkingDirectory=${PROJECT_DIR}
ExecStart=${PROJECT_DIR}/${PROJECT_NAME}
Restart=always
RestartSec=5
StandardOutput=append:/var/log/${PROJECT_NAME}/output.log
StandardError=append:/var/log/${PROJECT_NAME}/error.log

[Install]
WantedBy=multi-user.target
SERVICE

chown -R www-data:www-data ${PROJECT_DIR}
chmod +x ${PROJECT_DIR}/${PROJECT_NAME}
mkdir -p /var/log/${PROJECT_NAME}
chown www-data:www-data /var/log/${PROJECT_NAME}

systemctl daemon-reload
systemctl enable ${SERVICE_NAME}
systemctl restart ${SERVICE_NAME}

echo -e "${GREEN}✅ Деплой завершен!${NC}"
echo -e "${GREEN}🌐 Сайт: https://${DOMAIN}${NC}"
echo -e "${GREEN}📊 Логи: sudo journalctl -u ${SERVICE_NAME} -f${NC}"
echo -e "${GREEN}🔐 Админ: логин 'admin', пароль который вы указали${NC}"

# Проверка статуса
sleep 2
systemctl status ${SERVICE_NAME} --no-pager
