package main

import (
	"devops-manual/internal/database"
	"devops-manual/internal/handlers"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()

	// Подключение к БД
	db, err := database.New()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Инициализация схемы
	if err := db.InitSchema(); err != nil {
		log.Fatal("Failed to init schema:", err)
	}

	// Настройка Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.LoadHTMLGlob("web/templates/*")

	// Инициализация обработчиков
	h := handlers.New(db)
	h.RegisterRoutes(r)

	// Запуск мониторинга
	h.Monitor.StartMonitoring(5 * time.Minute)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
