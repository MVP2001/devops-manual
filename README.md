# DevOps Manual

Практический мануал по DevOps с возможностью создания и редактирования лабораторных работ.

## Быстрый деплой на новый сервер

```bash
# На новом сервере Ubuntu 22.04
sudo apt update && sudo apt install -y git

# Клонируй репозиторий
git clone https://github.com/mvp2001/devops-manual.git
cd devops-manual

# Запусти деплой
sudo ./deploy.sh
