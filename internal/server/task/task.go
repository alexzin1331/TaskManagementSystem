package server

import (
	"context"
	"errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"mod1/internal/models"
	auth "mod1/internal/server/auth"
	service "mod1/internal/services/task"
	"mod1/internal/storage"
	taskv1 "mod1/proto/gen/go"
	"time"
)

type TaskServer struct {
	taskv1.UnimplementedTaskServiceServer
	Service *service.TaskService
}

var ErrTaskNotFound = errors.New("Task not found")

func RegisterTaskServer(gRPCServer *grpc.Server, taskService *service.TaskService) {
	taskv1.RegisterTaskServiceServer(gRPCServer, &TaskServer{Service: taskService})
}

func (s *TaskServer) CreateTask(ctx context.Context, req *taskv1.CreateTaskRequest) (*taskv1.CreateTaskResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	var dueDate time.Time
	if req.DueDate != nil {
		dueDate = req.DueDate.AsTime()
	}

	taskID, err := s.Service.CreateTask(ctx, userID, req.Title, req.Description, dueDate, models.TaskStatus(taskv1.TaskStatus_TASK_STATUS_OPEN))
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create task")
	}

	task, err := s.Service.GetTask(ctx, userID, taskID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get created task")
	}

	return &taskv1.CreateTaskResponse{
		Task: convertTaskToProto(task),
	}, nil
}

func (s *TaskServer) GetTask(ctx context.Context, req *taskv1.GetTaskRequest) (*taskv1.GetTaskResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	task, err := s.Service.GetTask(ctx, userID, req.Id)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, status.Error(codes.Internal, "failed to get task")
	}

	return &taskv1.GetTaskResponse{
		Task: convertTaskToProto(task),
	}, nil
}

func (s *TaskServer) UpdateTask(ctx context.Context, req *taskv1.UpdateTaskRequest) (*taskv1.UpdateTaskResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	var dueDate time.Time
	if req.DueDate != nil {
		dueDate = req.DueDate.AsTime()
	}

	err = s.Service.UpdateTask(ctx, userID, req.Id, req.Title, req.Description, dueDate, models.TaskStatus(req.Status))
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, status.Error(codes.Internal, "failed to update task")
	}

	task, err := s.Service.GetTask(ctx, userID, req.Id)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to get updated task")
	}

	return &taskv1.UpdateTaskResponse{
		Task: convertTaskToProto(task),
	}, nil
}

func (s *TaskServer) DeleteTask(ctx context.Context, req *taskv1.DeleteTaskRequest) (*taskv1.DeleteTaskResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	err = s.Service.DeleteTask(ctx, userID, req.Id)
	if err != nil {
		if errors.Is(err, ErrTaskNotFound) {
			return nil, status.Error(codes.NotFound, "task not found")
		}
		return nil, status.Error(codes.Internal, "failed to delete task")
	}

	return &taskv1.DeleteTaskResponse{Success: true}, nil
}

func (s *TaskServer) ListTasks(ctx context.Context, req *taskv1.ListTasksRequest) (*taskv1.ListTasksResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	var dueDateFrom, dueDateTo time.Time
	if req.DueDateFrom != nil {
		dueDateFrom = req.DueDateFrom.AsTime()
	}
	if req.DueDateTo != nil {
		dueDateTo = req.DueDateTo.AsTime()
	}

	tasks, err := s.Service.ListTasks(ctx, userID, (*models.TaskStatus)(&req.Status), &dueDateFrom, &dueDateTo, req.PageSize, req.PageToken)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list tasks")
	}

	protoTasks := make([]*taskv1.Task, 0, len(tasks))
	for _, task := range tasks {
		protoTasks = append(protoTasks, convertTaskToProto(task))
	}

	return &taskv1.ListTasksResponse{
		Tasks:         protoTasks,
		NextPageToken: req.PageToken + 1,
	}, nil
}

func (s *TaskServer) SearchTasks(ctx context.Context, req *taskv1.SearchTasksRequest) (*taskv1.SearchTasksResponse, error) {
	userID, err := auth.GetUserIDFromContext(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "unauthorized")
	}

	tasks, err := s.Service.SearchTasks(ctx, userID, req.Query, req.PageSize, req.PageToken)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to search tasks")
	}

	protoTasks := make([]*taskv1.Task, 0, len(tasks))
	for _, task := range tasks {
		protoTasks = append(protoTasks, convertTaskToProto(task))
	}

	return &taskv1.SearchTasksResponse{
		Tasks:         protoTasks,
		NextPageToken: req.PageToken + 1,
	}, nil
}

func convertTaskToProto(task *storage.Task) *taskv1.Task {
	var dueDate, createdAt, updatedAt *timestamppb.Timestamp
	if task.DueDate != nil {
		dueDate = timestamppb.New(*task.DueDate)
	}
	if !task.CreatedAt.IsZero() {
		createdAt = timestamppb.New(task.CreatedAt)
	}
	if !task.UpdatedAt.IsZero() {
		updatedAt = timestamppb.New(task.UpdatedAt)
	}

	return &taskv1.Task{
		Id:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		DueDate:     dueDate,
		Status:      taskv1.TaskStatus(task.Status),
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
	}
}
