package main

import (
	"devops-manual/internal/database"
	"devops-manual/internal/handlers"
	"flag"
	"html/template"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Флаг для создания админа
	createAdmin := flag.Bool("create-admin", false, "Create admin user")
	flag.Parse()

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

	// Создание админа и выход
	if *createAdmin {
		password := os.Getenv("ADMIN_PASSWORD")
		if password == "" {
			password = "admin123"
		}
		if err := db.CreateUser("admin", password, true); err != nil {
			log.Println("Admin creation error (may already exist):", err)
		} else {
			log.Println("✅ Admin created successfully")
		}
		return
	}

	// Настройка Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	
	// Загрузка шаблонов с правильными именами
	tmpl := template.New("")
	
	files := map[string]string{
		"index.html":       "web/templates/index.html",
		"topic/topic.html": "web/templates/topic/topic.html",
		"lab/lab.html":     "web/templates/lab/lab.html",
		"auth/login.html":  "web/templates/auth/login.html",
	}
	
	for name, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			log.Fatal("Failed to read template", name, ":", err)
		}
		tmpl, err = tmpl.New(name).Parse(string(content))
		if err != nil {
			log.Fatal("Failed to parse template", name, ":", err)
		}
	}
	
	r.SetHTMLTemplate(tmpl)

	// Инициализация обработчиков
	h := handlers.New(db)
	h.RegisterRoutes(r)

	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
