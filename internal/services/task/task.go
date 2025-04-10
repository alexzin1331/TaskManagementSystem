package service

import (
	"context"
	"mod1/internal/models"
	"mod1/internal/storage"
	"time"
)

type TaskService struct {
	storage *storage.Storage
}

func NewTaskService(storage *storage.Storage) *TaskService {
	return &TaskService{storage: storage}
}

func (s *TaskService) CreateTask(ctx context.Context, userID int64, title, description string, dueDate time.Time, status models.TaskStatus) (int64, error) {
	var dueDateStr string
	if !dueDate.IsZero() {
		dueDateStr = dueDate.Format(time.RFC3339)
	}
	return s.storage.CreateTask(ctx, userID, title, description, dueDateStr, int32(status))
}

func (s *TaskService) GetTask(ctx context.Context, userID, taskID int64) (*storage.Task, error) {
	return s.storage.GetTask(ctx, userID, taskID)
}

func (s *TaskService) UpdateTask(ctx context.Context, userID, taskID int64, title, description string, dueDate time.Time, status models.TaskStatus) error {
	var dueDateStr string
	if !dueDate.IsZero() {
		dueDateStr = dueDate.Format(time.RFC3339)
	}
	return s.storage.UpdateTask(ctx, taskID, userID, title, description, dueDateStr, int32(status))
}

func (s *TaskService) DeleteTask(ctx context.Context, userID, taskID int64) error {
	return s.storage.DeleteTask(ctx, taskID, userID)
}

func (s *TaskService) ListTasks(ctx context.Context, userID int64, status *models.TaskStatus, dueDateFrom, dueDateTo *time.Time, pageSize, pageToken int32) ([]*storage.Task, error) {
	var statusInt *int32
	if status != nil {
		s := int32(*status)
		statusInt = &s
	}

	var fromStr, toStr *string
	if dueDateFrom != nil {
		s := dueDateFrom.Format(time.RFC3339)
		fromStr = &s
	}
	if dueDateTo != nil {
		s := dueDateTo.Format(time.RFC3339)
		toStr = &s
	}

	return s.storage.ListTasks(ctx, userID, statusInt, fromStr, toStr, pageSize, pageToken)
}

func (s *TaskService) SearchTasks(ctx context.Context, userID int64, query string, pageSize, pageToken int32) ([]*storage.Task, error) {
	return s.storage.SearchTasks(ctx, userID, query, pageSize, pageToken)
}
