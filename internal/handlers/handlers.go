package handlers

import (
	"devops-manual/internal/database"
	"devops-manual/internal/models"
	"devops-manual/internal/monitoring"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	DB       *database.DB
	Monitor  *monitoring.Monitor
}

func New(db *database.DB) *Handler {
	return &Handler{
		DB:      db,
		Monitor: monitoring.NewMonitor(db),
	}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	r.GET("/", h.Index)
	r.GET("/api/topics", h.GetTopics)
	r.GET("/api/labs/:topic", h.GetLabsByTopic)
	r.POST("/api/labs", h.CreateLab)
	r.GET("/api/metrics", h.GetMetrics)
	r.GET("/health", h.HealthCheck)
	
	// Статические файлы
	r.Static("/static", "./web/static")
}

func (h *Handler) Index(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "DevOps Manual",
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

	// Уведомление о новой лабе
	h.Monitor.SendAlert(fmt.Sprintf("📝 Новая лаба создана: %s", lab.Title))
	
	c.JSON(http.StatusCreated, lab)
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
