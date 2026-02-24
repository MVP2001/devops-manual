package database

import (
	"database/sql"
	"devops-manual/internal/models"
	"fmt"
	"os"
	"time"

	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

type DB struct {
	*sql.DB
	sessions map[string]*models.Session
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

	return &DB{db, make(map[string]*models.Session)}, nil
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
		topic_id INTEGER REFERENCES topics(id) ON DELETE CASCADE,
		title VARCHAR(255) NOT NULL,
		slug VARCHAR(255) NOT NULL,
		content TEXT,
		commands TEXT[],
		difficulty VARCHAR(50),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(topic_id, slug)
	);

	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(255) UNIQUE NOT NULL,
		password_hash VARCHAR(255) NOT NULL,
		is_admin BOOLEAN DEFAULT false,
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

	INSERT INTO users (username, password_hash, is_admin) 
	VALUES ('admin', '$2a$10$YourHashedPasswordHere', true)
	ON CONFLICT DO NOTHING;
	`

	_, err := db.Exec(schema)
	return err
}

func (db *DB) GetTopics() ([]models.Topic, error) {
	rows, err := db.Query("SELECT id, title, slug, description, created_at FROM topics ORDER BY title")
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

func (db *DB) GetTopicBySlug(slug string) (*models.Topic, error) {
	var t models.Topic
	err := db.QueryRow("SELECT id, title, slug, description, created_at FROM topics WHERE slug = $1", slug).
		Scan(&t.ID, &t.Title, &t.Slug, &t.Description, &t.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (db *DB) GetLabsByTopicSlug(topicSlug string) ([]models.Lab, error) {
	query := `
		SELECT l.id, l.topic_id, l.title, l.slug, l.content, l.commands, l.difficulty, l.created_at, l.updated_at,
		       t.id, t.title, t.slug, t.description, t.created_at
		FROM labs l
		JOIN topics t ON l.topic_id = t.id
		WHERE t.slug = $1
		ORDER BY l.created_at DESC`

	rows, err := db.Query(query, topicSlug)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var labs []models.Lab
	for rows.Next() {
		var l models.Lab
		var t models.Topic
		err := rows.Scan(
			&l.ID, &l.TopicID, &l.Title, &l.Slug, &l.Content, pq.Array(&l.Commands), &l.Difficulty, &l.CreatedAt, &l.UpdatedAt,
			&t.ID, &t.Title, &t.Slug, &t.Description, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		l.Topic = &t
		labs = append(labs, l)
	}
	return labs, nil
}

func (db *DB) GetLabBySlug(topicSlug, labSlug string) (*models.Lab, error) {
	query := `
		SELECT l.id, l.topic_id, l.title, l.slug, l.content, l.commands, l.difficulty, l.created_at, l.updated_at,
		       t.id, t.title, t.slug, t.description, t.created_at
		FROM labs l
		JOIN topics t ON l.topic_id = t.id
		WHERE t.slug = $1 AND l.slug = $2`

	var l models.Lab
	var t models.Topic
	err := db.QueryRow(query, topicSlug, labSlug).Scan(
		&l.ID, &l.TopicID, &l.Title, &l.Slug, &l.Content, pq.Array(&l.Commands), &l.Difficulty, &l.CreatedAt, &l.UpdatedAt,
		&t.ID, &t.Title, &t.Slug, &t.Description, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	l.Topic = &t
	return &l, nil
}

func (db *DB) CreateLab(lab *models.Lab) error {
	// Генерируем slug из названия
	lab.Slug = slugify(lab.Title)
	
	query := `INSERT INTO labs (topic_id, title, slug, content, commands, difficulty) 
	          VALUES ($1, $2, $3, $4, $5, $6) 
	          RETURNING id, created_at, updated_at`
	
	err := db.QueryRow(query, lab.TopicID, lab.Title, lab.Slug, lab.Content,
		pq.Array(lab.Commands), lab.Difficulty).Scan(&lab.ID, &lab.CreatedAt, &lab.UpdatedAt)
	
	return err
}

func (db *DB) UpdateLab(lab *models.Lab) error {
	query := `UPDATE labs 
	          SET title = $1, content = $2, commands = $3, difficulty = $4, updated_at = NOW()
	          WHERE id = $5`
	
	_, err := db.Exec(query, lab.Title, lab.Content, pq.Array(lab.Commands), lab.Difficulty, lab.ID)
	return err
}

func (db *DB) DeleteLab(id int) error {
	_, err := db.Exec("DELETE FROM labs WHERE id = $1", id)
	return err
}

// Auth methods
func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	var u models.User
	err := db.QueryRow("SELECT id, username, password_hash, is_admin FROM users WHERE username = $1", username).
		Scan(&u.ID, &u.Username, &u.Password, &u.IsAdmin)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (db *DB) CreateSession(userID int) string {
	token := generateToken()
	db.sessions[token] = &models.Session{
		Token:     token,
		UserID:    userID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}
	return token
}

func (db *DB) GetSession(token string) *models.Session {
	session, exists := db.sessions[token]
	if !exists || session.ExpiresAt.Before(time.Now()) {
		return nil
	}
	return session
}

func (db *DB) DeleteSession(token string) {
	delete(db.sessions, token)
}

func (db *DB) CreateUser(username, password string, isAdmin bool) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	_, err = db.Exec("INSERT INTO users (username, password_hash, is_admin) VALUES ($1, $2, $3)",
		username, string(hash), isAdmin)
	return err
}

// Helpers
func slugify(s string) string {
	// Простая реализация - заменить пробелы на дефисы и привести к нижнему регистру
	// В продакшене лучше использовать библиотеку
	result := ""
	for _, r := range s {
		if r == ' ' {
			result += "-"
		} else if r >= 'A' && r <= 'Z' {
			result += string(r + 32)
		} else if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result += string(r)
		}
	}
	return result
}

func generateToken() string {
	// Простая генерация токена
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().Unix())
}
