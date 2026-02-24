#!/bin/bash
set -e

echo "🚀 Deploying DevOps Manual..."

# Переменные
PROJECT_DIR="/var/www/devops-manual"
SERVICE_NAME="devops-manual"
GO_BIN="/usr/local/go/bin/go"

# Проверка Go
if ! command -v $GO_BIN &> /dev/null; then
    echo "❌ Go не найден в /usr/local/go/bin/"
    echo "Установи Go: wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz  && sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz"
    exit 1
fi

echo "✅ Go найден: $($GO_BIN version)"

# Создание директории
sudo mkdir -p $PROJECT_DIR
sudo chown -R $USER:$USER $PROJECT_DIR

# Копирование файлов
if [ ! -d "$PROJECT_DIR/.git" ]; then
    git clone https://github.com/mvp2001/devops-manual.git  $PROJECT_DIR
else
    cd $PROJECT_DIR && git pull
fi

# Установка зависимостей Go
cd $PROJECT_DIR
$GO_BIN mod tidy
$GO_BIN mod download

# Сборка бинарника (лучше чем go run для production)
$GO_BIN build -o devops-manual cmd/main.go

# Копирование .env
if [ ! -f "$PROJECT_DIR/.env" ]; then
    echo "⚠️ Создайте файл .env в $PROJECT_DIR!"
    exit 1
fi

# Настройка прав
sudo chown -R www-data:www-data $PROJECT_DIR
sudo chmod +x $PROJECT_DIR/devops-manual

# Настройка Nginx
sudo cp deployments/nginx.conf /etc/nginx/sites-available/devops-manual
sudo ln -sf /etc/nginx/sites-available/devops-manual /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t && sudo systemctl reload nginx

# SSL
sudo certbot --nginx -d mvp2001.ru -d www.mvp2001.ru --non-interactive --agree-tos -m mihailpodorets01@gmail.com || true

# Обновленный systemd сервис (используем бинарник вместо go run)
sudo tee /etc/systemd/system/$SERVICE_NAME.service > /dev/null <<EOF
[Unit]
Description=DevOps Manual Web Service
After=network.target postgresql.service

[Service]
Type=simple
User=www-data
WorkingDirectory=$PROJECT_DIR
ExecStart=$PROJECT_DIR/devops-manual
Restart=always
RestartSec=5
Environment=GO_ENV=production

[Install]
WantedBy=multi-user.target
