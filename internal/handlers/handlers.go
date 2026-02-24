package handlers

import (
	"devops-manual/internal/database"
	"devops-manual/internal/models"
	"devops-manual/internal/monitoring"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	DB       *database.DB
	Monitor  *monitoring.Monitor
}

func New(db *database.DB) *Handler {
	return &Handler{
		DB:      db,
		Monitor: monitoring.NewMonitor(db.DB),
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	// Главная
	r.GET("/", h.Index)
	
	// API
	r.GET("/api/topics", h.GetTopics)
	r.GET("/api/topics/:slug", h.GetTopicAPI)
	r.GET("/api/topics/:slug/labs", h.GetLabsAPI)
	r.GET("/api/labs/:topic/:lab", h.GetLabAPI)
	r.POST("/api/labs", h.AuthMiddleware(), h.CreateLab)
	r.PUT("/api/labs/:id", h.AuthMiddleware(), h.UpdateLab)
	r.DELETE("/api/labs/:id", h.AuthMiddleware(), h.DeleteLab)
	r.GET("/api/metrics", h.GetMetrics)
	
	// Auth API
	r.POST("/api/auth/login", h.Login)
	r.POST("/api/auth/logout", h.Logout)
	r.GET("/api/auth/check", h.CheckAuth)
	
	// HTML страницы - БЕЗ КОНФЛИКТОВ
	r.GET("/topic/:slug", h.TopicPage)           // /topic/docker
	r.GET("/lab/:topic/:lab", h.LabPage)         // /lab/docker/container-run
	r.GET("/login", h.LoginPage)
	
	// Health
	r.GET("/health", h.HealthCheck)
	
	// Статические файлы
	r.Static("/static", "./web/static")
}

func (h *Handler) Index(c *gin.Context) {
	topics, _ := h.DB.GetTopics()
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title":  "DevOps Manual",
		"topics": topics,
	})
}

func (h *Handler) GetTopics(c *gin.Context) {
	topics, err := h.DB.GetTopics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, topics)
}

func (h *Handler) GetTopicAPI(c *gin.Context) {
	slug := c.Param("slug")
	topic, err := h.DB.GetTopicBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Topic not found"})
		return
	}
	c.JSON(http.StatusOK, topic)
}

func (h *Handler) GetLabsAPI(c *gin.Context) {
	slug := c.Param("slug")
	labs, err := h.DB.GetLabsByTopicSlug(slug)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, labs)
}

func (h *Handler) GetLabAPI(c *gin.Context) {
	topicSlug := c.Param("topic")
	labSlug := c.Param("lab")
	lab, err := h.DB.GetLabBySlug(topicSlug, labSlug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Lab not found"})
		return
	}
	c.JSON(http.StatusOK, lab)
}

func (h *Handler) CreateLab(c *gin.Context) {
	var lab models.Lab
	if err := c.ShouldBindJSON(&lab); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.DB.CreateLab(&lab); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.Monitor.SendAlert(fmt.Sprintf("📝 Новая лаба создана: %s", lab.Title))
	c.JSON(http.StatusCreated, lab)
}

func (h *Handler) UpdateLab(c *gin.Context) {
	var lab models.Lab
	if err := c.ShouldBindJSON(&lab); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, _ := strconv.Atoi(c.Param("id"))
	lab.ID = id
	
	if err := h.DB.UpdateLab(&lab); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, lab)
}

func (h *Handler) DeleteLab(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := h.DB.DeleteLab(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Deleted"})
}

// HTML Pages
func (h *Handler) TopicPage(c *gin.Context) {
	slug := c.Param("slug")
	topic, err := h.DB.GetTopicBySlug(slug)
	if err != nil {
		c.HTML(http.StatusNotFound, "404.html", nil)
		return
	}

	labs, _ := h.DB.GetLabsByTopicSlug(slug)
	
	c.HTML(http.StatusOK, "topic/topic.html", gin.H{
		"title": topic.Title,
		"topic": topic,
		"labs":  labs,
	})
}

func (h *Handler) LabPage(c *gin.Context) {
	topicSlug := c.Param("topic")
	labSlug := c.Param("lab")
	
	lab, err := h.DB.GetLabBySlug(topicSlug, labSlug)
	if err != nil {
		c.HTML(http.StatusNotFound, "404.html", nil)
		return
	}

	c.HTML(http.StatusOK, "lab/lab.html", gin.H{
		"title": lab.Title,
		"lab":   lab,
	})
}

func (h *Handler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/login.html", gin.H{
		"title": "Login",
	})
}

// Auth handlers
func (h *Handler) Login(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user, err := h.DB.GetUserByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token := h.DB.CreateSession(user.ID)
	c.SetCookie("session", token, 86400, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"user": user.Username, "is_admin": user.IsAdmin})
}

func (h *Handler) Logout(c *gin.Context) {
	token, _ := c.Cookie("session")
	h.DB.DeleteSession(token)
	c.SetCookie("session", "", -1, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

func (h *Handler) CheckAuth(c *gin.Context) {
	token, err := c.Cookie("session")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"authenticated": false})
		return
	}

	session := h.DB.GetSession(token)
	if session == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"authenticated": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"authenticated": true})
}

func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		session := h.DB.GetSession(token)
		if session == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			c.Abort()
			return
		}

		c.Set("user_id", session.UserID)
		c.Next()
	}
}

func (h *Handler) GetMetrics(c *gin.Context) {
	metrics, err := h.Monitor.GetMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

func (h *Handler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}
