package server

import (
	"context"
	"log"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var publicMethods = map[string]bool{
	"/task_service.TaskService/Login":    true,
	"/task_service.TaskService/Register": true,
}

func AuthInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	log.Printf("Incoming call: %s", info.FullMethod)

	publicMethods := map[string]bool{
		"/task_service.AuthService/Login":    true,
		"/task_service.AuthService/Register": true,
	}

	if publicMethods[info.FullMethod] {
		log.Printf("Public method %s, skipping auth", info.FullMethod)
		return handler(ctx, req)
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authorization token is not provided")
	}

	token := strings.TrimPrefix(authHeader[0], "Bearer ")
	if token == authHeader[0] {
		return nil, status.Error(codes.Unauthenticated, "invalid authorization format")
	}

	userID, err := verifyToken(token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid token")
	}

	ctx = context.WithValue(ctx, "userID", userID)

	return handler(ctx, req)
}
