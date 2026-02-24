#!/bin/bash
set -e

echo "🚀 Deploying DevOps Manual..."

# Переменные
PROJECT_DIR="/var/www/devops-manual"
SERVICE_NAME="devops-manual"

# Создание директории
sudo mkdir -p $PROJECT_DIR
sudo chown -R $USER:$USER $PROJECT_DIR

# Копирование файлов (предполагается, что код уже в git)
if [ ! -d "$PROJECT_DIR/.git" ]; then
    git clone https://github.com/mvp2001/devops-manual.git $PROJECT_DIR
else
    cd $PROJECT_DIR && git pull
fi

# Установка зависимостей Go
cd $PROJECT_DIR
go mod tidy
go mod download

# Копирование .env (должен быть создан вручную!)
if [ ! -f "$PROJECT_DIR/.env" ]; then
    echo "⚠️ Создайте файл .env в $PROJECT_DIR!"
    exit 1
fi

# Настройка Nginx
sudo cp deployments/nginx.conf /etc/nginx/sites-available/devops-manual
sudo ln -sf /etc/nginx/sites-available/devops-manual /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx

# SSL (Let's Encrypt)
sudo certbot --nginx -d mvp2001.ru -d www.mvp2001.ru --non-interactive --agree-tos -m mihailpodorets01@gmail.com

# Systemd сервис
sudo cp deployments/devops-manual.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable $SERVICE_NAME
sudo systemctl restart $SERVICE_NAME

echo "✅ Deploy completed!"
echo "🌐 Site: https://mvp2001.ru"
echo "📊 Metrics: https://mvp2001.ru/api/metrics"
