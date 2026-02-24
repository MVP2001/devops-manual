package models

import "time"

type Topic struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type Lab struct {
	ID          int       `json:"id"`
	TopicID     int       `json:"topic_id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Commands    []string  `json:"commands"`
	Difficulty  string    `json:"difficulty"`
	CreatedAt   time.Time `json:"created_at"`
}

type SystemMetrics struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	DiskUsage   float64 `json:"disk_usage"`
	Timestamp   int64   `json:"timestamp"`
}
