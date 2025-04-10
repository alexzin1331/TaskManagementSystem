package client

import (
	"context"
	"fmt"
	"google.golang.org/grpc/metadata"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
	taskv1 "mod1/proto/gen/go"
)

type TaskClient struct {
	authClient taskv1.AuthServiceClient // новый клиент
	taskClient taskv1.TaskServiceClient
	conn       *grpc.ClientConn
	token      string
}

func NewTaskClient(addr string) (*TaskClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &TaskClient{
		authClient: taskv1.NewAuthServiceClient(conn), // новый клиент
		taskClient: taskv1.NewTaskServiceClient(conn),
		conn:       conn,
	}, nil
}

func (c *TaskClient) SetToken(token string) {
	c.token = token
}

func (c *TaskClient) Close() error {
	return c.conn.Close()
}

func (c *TaskClient) withAuth(ctx context.Context) context.Context {
	if c.token == "" {
		return ctx
	}

	md := metadata.Pairs("authorization", "Bearer "+c.token)
	return metadata.NewOutgoingContext(ctx, md)
}

func (c *TaskClient) Register(ctx context.Context, username, email, password string) (int64, error) {
	fmt.Println("client-Register")
	resp, err := c.authClient.Register(ctx, &taskv1.RegisterRequest{
		Username: username,
		Email:    email,
		Password: password,
	})
	if err != nil {
		log.Printf("Register failed: %v", err)
		return 0, err
	}
	return resp.UserId, nil
}

func (c *TaskClient) Login(ctx context.Context, email, password string) (string, error) {
	resp, err := c.authClient.Login(ctx, &taskv1.LoginRequest{ // изменилось
		Email:    email,
		Password: password,
	})
	if err != nil {
		log.Printf("Login failed: %v", err)
		return "", err
	}
	c.token = resp.Token
	return resp.Token, nil
}

func (c *TaskClient) CreateTask(ctx context.Context, title, description string, dueDate time.Time) (*taskv1.Task, error) {

	resp, err := c.taskClient.CreateTask(c.withAuth(ctx), &taskv1.CreateTaskRequest{
		Title:       title,
		Description: description,
		DueDate:     timestamppb.New(dueDate),
	})
	if err != nil {
		log.Printf("CreateTask failed: %v", err)
		return nil, err
	}
	return resp.Task, nil
}

func (c *TaskClient) GetTask(ctx context.Context, id int64) (*taskv1.Task, error) {
	resp, err := c.taskClient.GetTask(c.withAuth(ctx), &taskv1.GetTaskRequest{Id: id})
	if err != nil {
		log.Printf("GetTask failed: %v", err)
		return nil, err
	}
	return resp.Task, nil
}

func (c *TaskClient) UpdateTask(ctx context.Context, id int64, title, description string, dueDate time.Time, status taskv1.TaskStatus) (*taskv1.Task, error) {
	resp, err := c.taskClient.UpdateTask(c.withAuth(ctx), &taskv1.UpdateTaskRequest{
		Id:          id,
		Title:       title,
		Description: description,
		DueDate:     timestamppb.New(dueDate),
		Status:      status,
	})
	if err != nil {
		log.Printf("UpdateTask failed: %v", err)
		return nil, err
	}
	return resp.Task, nil
}

func (c *TaskClient) DeleteTask(ctx context.Context, id int64) (bool, error) {
	resp, err := c.taskClient.DeleteTask(c.withAuth(ctx), &taskv1.DeleteTaskRequest{Id: id})
	if err != nil {
		log.Printf("DeleteTask failed: %v", err)
		return false, err
	}
	return resp.Success, nil
}

func (c *TaskClient) ListTasks(ctx context.Context, status taskv1.TaskStatus, dueDateFrom, dueDateTo time.Time, pageSize, pageToken int32) ([]*taskv1.Task, int32, error) {
	resp, err := c.taskClient.ListTasks(c.withAuth(ctx), &taskv1.ListTasksRequest{
		Status:      status,
		DueDateFrom: timestamppb.New(dueDateFrom),
		DueDateTo:   timestamppb.New(dueDateTo),
		PageSize:    pageSize,
		PageToken:   pageToken,
	})
	if err != nil {
		log.Printf("ListTasks failed: %v", err)
		return nil, 0, err
	}
	return resp.Tasks, resp.NextPageToken, nil
}

func (c *TaskClient) SearchTasks(ctx context.Context, query string, pageSize, pageToken int32) ([]*taskv1.Task, int32, error) {
	resp, err := c.taskClient.SearchTasks(c.withAuth(ctx), &taskv1.SearchTasksRequest{
		Query:     query,
		PageSize:  pageSize,
		PageToken: pageToken,
	})
	if err != nil {
		log.Printf("SearchTasks failed: %v", err)
		return nil, 0, err
	}
	return resp.Tasks, resp.NextPageToken, nil
}
