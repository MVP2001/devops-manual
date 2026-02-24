package database

import (
	"database/sql"
	"devops-manual/internal/models"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

type DB struct {
	*sql.DB
}

func New() (*DB, error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

func (db *DB) InitSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS topics (
		id SERIAL PRIMARY KEY,
		title VARCHAR(255) NOT NULL,
		slug VARCHAR(255) UNIQUE NOT NULL,
		description TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS labs (
		id SERIAL PRIMARY KEY,
		topic_id INTEGER REFERENCES topics(id),
		title VARCHAR(255) NOT NULL,
		content TEXT,
		commands TEXT[],
		difficulty VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS system_logs (
		id SERIAL PRIMARY KEY,
		level VARCHAR(50),
		message TEXT,
		metrics JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	INSERT INTO topics (title, slug, description) VALUES 
		('Docker', 'docker', 'Контейнеризация приложений'),
		('Kubernetes', 'kubernetes', 'Оркестрация контейнеров'),
		('CI/CD', 'cicd', 'Непрерывная интеграция и доставка'),
		('Terraform', 'terraform', 'Инфраструктура как код'),
		('Monitoring', 'monitoring', 'Мониторинг и логирование')
	ON CONFLICT DO NOTHING;
	`

	_, err := db.Exec(schema)
	return err
}

func (db *DB) GetTopics() ([]models.Topic, error) {
	rows, err := db.Query("SELECT id, title, slug, description, created_at FROM topics ORDER BY created_at")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []models.Topic
	for rows.Next() {
		var t models.Topic
		err := rows.Scan(&t.ID, &t.Title, &t.Slug, &t.Description, &t.CreatedAt)
		if err != nil {
			return nil, err
		}
		topics = append(topics, t)
	}
	return topics, nil
}

func (db *DB) CreateLab(lab *models.Lab) error {
	query := `INSERT INTO labs (topic_id, title, content, commands, difficulty) 
	          VALUES ($1, $2, $3, $4, $5) RETURNING id, created_at`
	return db.QueryRow(query, lab.TopicID, lab.Title, lab.Content, 
		pq.Array(lab.Commands), lab.Difficulty).Scan(&lab.ID, &lab.CreatedAt)
}
