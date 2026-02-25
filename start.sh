#!/bin/bash

set -e

echo "🚀 Запуск DevOps Manual в Docker..."

# Проверка наличия .env
if [ ! -f .env ]; then
    echo "⚠️  Файл .env не найден!"
    echo "Создаю из .env.docker..."
    if [ -f .env.docker ]; then
        cp .env.docker .env
        echo "❗ Отредактируй .env и запусти скрипт снова"
        exit 1
    else
        echo "❌ .env.docker тоже не найден"
        exit 1
    fi
fi

# Загрузка переменных
export $(grep -v '^#' .env | xargs)

echo "📦 Запуск контейнеров..."
docker-compose up -d --build

echo "⏳ Ожидание инициализации..."
sleep 10

echo "🔧 Создание администратора..."
docker-compose exec -T app ./devops-manual -create-admin || true

echo "✅ Готово!"
echo "🌐 Сайт: http://localhost или https://${DOMAIN}"
echo "📊 Логи: docker-compose logs -f app"
echo ""
echo "Полезные команды:"
echo "  docker-compose ps        - статус контейнеров"
echo "  docker-compose logs -f   - смотреть логи"
echo "  docker-compose down      - остановить"
echo "  docker-compose down -v   - остановить и удалить данные"
