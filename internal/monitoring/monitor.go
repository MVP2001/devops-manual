package monitoring

import (
	"devops-manual/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/mem"
)

type Monitor struct {
	botToken string
	chatID   string
	db       *sql.DB
}

func NewMonitor(db *sql.DB) *Monitor {
	return &Monitor{
		botToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
		chatID:   os.Getenv("TELEGRAM_CHAT_ID"),
		db:       db,
	}
}

func (m *Monitor) GetMetrics() (*models.SystemMetrics, error) {
	cpuPercent, _ := cpu.Percent(time.Second, false)
	memInfo, _ := mem.VirtualMemory()
	diskInfo, _ := disk.Usage("/")

	return &models.SystemMetrics{
		CPUUsage:    cpuPercent[0],
		MemoryUsage: memInfo.UsedPercent,
		DiskUsage:   diskInfo.UsedPercent,
		Timestamp:   time.Now().Unix(),
	}, nil
}

func (m *Monitor) SendAlert(message string) error {
	if m.botToken == "" || m.chatID == "" {
		return fmt.Errorf("telegram credentials not configured")
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", m.botToken)
	payload := fmt.Sprintf(`{"chat_id": "%s", "text": "🚨 DevOps Manual Alert:\n%s", "parse_mode": "Markdown"}`, 
		m.chatID, message)

	resp, err := http.Post(url, "application/json", strings.NewReader(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Сохраняем в БД
	metrics, _ := m.GetMetrics()
	metricsJSON, _ := json.Marshal(metrics)
	
	_, err = m.db.Exec("INSERT INTO system_logs (level, message, metrics) VALUES ($1, $2, $3)",
		"ALERT", message, metricsJSON)
	
	return err
}

func (m *Monitor) CheckThresholds() {
	metrics, _ := m.GetMetrics()
	
	if metrics.CPUUsage > 80 {
		m.SendAlert(fmt.Sprintf("⚠️ High CPU Usage: %.2f%%", metrics.CPUUsage))
	}
	if metrics.MemoryUsage > 85 {
		m.SendAlert(fmt.Sprintf("⚠️ High Memory Usage: %.2f%%", metrics.MemoryUsage))
	}
	if metrics.DiskUsage > 90 {
		m.SendAlert(fmt.Sprintf("⚠️ High Disk Usage: %.2f%%", metrics.DiskUsage))
	}
}

func (m *Monitor) StartMonitoring(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			m.CheckThresholds()
			
			// Логирование метрик
			metrics, _ := m.GetMetrics()
			metricsJSON, _ := json.Marshal(metrics)
			m.db.Exec("INSERT INTO system_logs (level, message, metrics) VALUES ($1, $2, $3)",
				"INFO", "Routine metrics check", metricsJSON)
		}
	}()
}
