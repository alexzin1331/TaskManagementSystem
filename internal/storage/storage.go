package storage

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"mod1/internal/models"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	cfg "mod1/config"
)

const (
	migrationPath = "file://migrations"
)

type Task struct {
	ID          int64
	UserID      int64
	Title       string
	Description string
	DueDate     *time.Time
	Status      int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
type Storage struct {
	db *sql.DB
}

func New(c cfg.DatabaseCfg) (*Storage, error) {
	const op = "storage.connection"
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", c.Host, c.Port, c.User, c.Password, c.DBName)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}
	s := &Storage{
		db: db,
	}
	if err = RunMigrations(db); err != nil {
		return &Storage{}, fmt.Errorf("failed to make migrations")
	}

	return s, nil
}

func RunMigrations(db *sql.DB) error {
	const op = "storage.migrations"
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		migrationPath,
		"postgres",
		driver,
	)
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	if err = m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			return fmt.Errorf("%s: %w", op, err)
		}
		log.Println("No migrations to apply.")
	} else {
		log.Println("Database migrations applied successfully.")
	}

	return nil
}

func (s *Storage) Register(ctx context.Context, username, email string, passHash []byte) (int64, error) {
	const op = "storage.postgres.Register"

	stmt, err := s.db.PrepareContext(ctx,
		"INSERT INTO users (username, email, password_hash) VALUES ($1, $2, $3) RETURNING id")
	if err != nil {
		return 0, fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	var userID int64
	err = stmt.QueryRowContext(ctx, username, email, passHash).Scan(&userID)
	if err != nil {
		return 0, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return userID, nil
}

func (s *Storage) GetUserByUsername(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgres.GetUserByUsername"

	stmt, err := s.db.PrepareContext(ctx, "SELECT id, password_hash, username FROM users WHERE email = $1")
	if err != nil {
		return models.User{}, fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	var userID int64
	var passwordHash []byte
	username := ""
	err = stmt.QueryRowContext(ctx, email).Scan(&userID, &passwordHash, &username)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.User{}, fmt.Errorf("%s: user not found", op)
		}
		return models.User{}, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return models.User{
		ID:       userID,
		Username: username,
		Email:    email,
		PassHash: passwordHash,
	}, nil
}

func (s *Storage) CreateTask(ctx context.Context, userID int64, title, description string, dueDate string, status int32) (int64, error) {
	const op = "storage.postgres.CreateTask"

	var parsedDueDate time.Time
	var err error
	if dueDate != "" {
		parsedDueDate, err = time.Parse(time.RFC3339, dueDate)
		if err != nil {
			return 0, fmt.Errorf("%s: invalid due date format: %w", op, err)
		}
	}
	var nullableDueDate sql.NullTime
	if dueDate != "" {
		nullableDueDate.Time = parsedDueDate
		nullableDueDate.Valid = true
	} else {
		nullableDueDate.Valid = false
	}

	stmt, err := s.db.PrepareContext(ctx,
		"INSERT INTO tasks (user_id, title, description, due_date, status) VALUES ($1, $2, $3, $4, $5) RETURNING id")
	if err != nil {
		return 0, fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	var taskID int64
	err = stmt.QueryRowContext(ctx, userID, title, description, nullableDueDate, status).Scan(&taskID)
	if err != nil {
		return 0, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	return taskID, nil
}

func (s *Storage) GetTask(ctx context.Context, userID, taskID int64) (*Task, error) {
	const op = "storage.postgres.GetTask"

	stmt, err := s.db.PrepareContext(ctx,
		"SELECT id, user_id, title, description, due_date, status, created_at, updated_at FROM tasks WHERE id = $1 AND user_id = $2")
	if err != nil {
		return nil, fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	task := &Task{}
	var dueDate sql.NullTime
	err = stmt.QueryRowContext(ctx, taskID, userID).Scan(
		&task.ID, &task.UserID, &task.Title, &task.Description, &dueDate, &task.Status, &task.CreatedAt, &task.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("%s: task not found", op)
		}
		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}

	if dueDate.Valid {
		task.DueDate = &dueDate.Time
	}

	return task, nil
}

func (s *Storage) UpdateTask(ctx context.Context, taskID, userID int64, title, description string, dueDate string, status int32) error {
	const op = "storage.postgres.UpdateTask"

	var parsedDueDate sql.NullTime
	if dueDate != "" {
		t, err := time.Parse(time.RFC3339, dueDate)
		if err != nil {
			return fmt.Errorf("%s: invalid due date format: %w", op, err)
		}
		parsedDueDate = sql.NullTime{Time: t, Valid: true}
	} else {
		parsedDueDate = sql.NullTime{Valid: false}
	}

	stmt, err := s.db.PrepareContext(ctx,
		"UPDATE tasks SET title = $1, description = $2, due_date = $3, status = $4, updated_at = NOW() WHERE id = $5 AND user_id = $6")
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, title, description, parsedDueDate, status, taskID, userID)
	if err != nil {
		return fmt.Errorf("%s: execute statement: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: get rows affected: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: task not found or unauthorized", op)
	}

	return nil
}

func (s *Storage) DeleteTask(ctx context.Context, taskID, userID int64) error {
	const op = "storage.postgres.DeleteTask"

	stmt, err := s.db.PrepareContext(ctx, "DELETE FROM tasks WHERE id = $1 AND user_id = $2")
	if err != nil {
		return fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	res, err := stmt.ExecContext(ctx, taskID, userID)
	if err != nil {
		return fmt.Errorf("%s: execute statement: %w", op, err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("%s: get rows affected: %w", op, err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("%s: task not found or unauthorized", op)
	}

	return nil
}

func (s *Storage) ListTasks(ctx context.Context, userID int64, status *int32, dueDateFrom, dueDateTo *string, pageSize, pageToken int32) ([]*Task, error) {
	const op = "storage.postgres.ListTasks"

	query := "SELECT id, user_id, title, description, due_date, status, created_at, updated_at FROM tasks WHERE user_id = $1"
	var args []interface{}
	args = append(args, userID)
	argCount := 2

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, *status)
		argCount++
	}

	if dueDateFrom != nil {
		query += fmt.Sprintf(" AND due_date >= $%d", argCount)
		t, err := time.Parse(time.RFC3339, *dueDateFrom)
		if err != nil {
			return nil, fmt.Errorf("%s: invalid due date from format: %w", op, err)
		}
		args = append(args, t)
		argCount++
	}

	if dueDateTo != nil {
		query += fmt.Sprintf(" AND due_date <= $%d", argCount)

		t, err := time.Parse(time.RFC3339, *dueDateTo)
		if err != nil {
			return nil, fmt.Errorf("%s: invalid due date to format: %w", op, err)
		}
		args = append(args, t)
		argCount++
	}

	query += " LIMIT $" + fmt.Sprint(argCount) + " OFFSET $" + fmt.Sprint(argCount+1)
	args = append(args, pageSize)
	args = append(args, pageSize*pageToken)

	stmt, err := s.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		var dueDate sql.NullTime
		err := rows.Scan(
			&task.ID, &task.UserID, &task.Title, &task.Description, &dueDate, &task.Status, &task.CreatedAt, &task.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("%s: scan row: %w", op, err)
		}
		if dueDate.Valid {
			task.DueDate = &dueDate.Time
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows error: %w", op, err)
	}

	return tasks, nil
}

func (s *Storage) SearchTasks(ctx context.Context, userID int64, query string, pageSize, pageToken int32) ([]*Task, error) {
	const op = "storage.postgres.SearchTasks"

	stmt, err := s.db.PrepareContext(ctx,
		"SELECT id, user_id, title, description, due_date, status, created_at, updated_at FROM tasks "+
			"WHERE user_id = $1 AND (title ILIKE $2 OR description ILIKE $2) "+
			"LIMIT $3 OFFSET $4")
	if err != nil {
		return nil, fmt.Errorf("%s: prepare statement: %w", op, err)
	}
	defer stmt.Close()

	searchQuery := "%" + query + "%"

	rows, err := stmt.QueryContext(ctx, userID, searchQuery, pageSize, pageSize*pageToken)
	if err != nil {
		return nil, fmt.Errorf("%s: execute statement: %w", op, err)
	}
	defer rows.Close()

	var tasks []*Task
	for rows.Next() {
		task := &Task{}
		var dueDate sql.NullTime

		err := rows.Scan(&task.ID, &task.UserID, &task.Title, &task.Description, &dueDate, &task.Status, &task.CreatedAt, &task.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("%s: scan row: %w", op, err)
		}
		if dueDate.Valid {
			task.DueDate = &dueDate.Time
		}

		tasks = append(tasks, task)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: rows error: %w", op, err)
	}

	return tasks, nil
}
