package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	service "mod1/internal/services/auth"
	authv1 "mod1/proto/gen/go"
	taskv1 "mod1/proto/gen/go"
	"strings"
	"time"
)

const (
	tokenDuration = 1 * time.Hour
	secretKey     = "secret"
)

type AuthServer struct {
	authv1.UnimplementedAuthServiceServer // изменилось
	AuthService                           *service.Auth
}

func RegisterAuthServer(gRPCServer *grpc.Server, authService *service.Auth) {
	authv1.RegisterAuthServiceServer(gRPCServer, &AuthServer{AuthService: authService}) // изменилось
}

func (s *AuthServer) Register(ctx context.Context, req *taskv1.RegisterRequest) (*taskv1.RegisterResponse, error) {
	if req.Username == "" {
		return nil, status.Error(codes.InvalidArgument, "username is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}

	userID, err := s.AuthService.RegisterNewUser(ctx, req.Username, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			return nil, status.Error(codes.AlreadyExists, "user already exists")
		}
		return nil, status.Error(codes.Internal, "failed to register user")
	}

	return &taskv1.RegisterResponse{UserId: userID}, nil
}

func (s *AuthServer) Login(ctx context.Context, req *taskv1.LoginRequest) (*taskv1.LoginResponse, error) {
	if req.Email == "" {
		return nil, status.Error(codes.InvalidArgument, "email is required")
	}
	if req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "password is required")
	}
	token, err := s.AuthService.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return nil, status.Error(codes.Unauthenticated, "invalid credentials")
		}
		return nil, status.Error(codes.Internal, "failed to login")
	}

	return &taskv1.LoginResponse{
		Token: token,
	}, nil
}

// Функция для создания JWT токена
/*func createToken(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(tokenDuration).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secretKey))
}*/

// Функция для проверки JWT токена
func verifyToken(tokenString string) (int64, error) {

	if tokenString == "" {
		return 0, fmt.Errorf("empty token string")
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		return 0, fmt.Errorf("token parsing failed: %w", err)
	}

	if !token.Valid {
		return 0, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, fmt.Errorf("invalid token claims format")
	}

	userIDClaim, exists := claims["user_id"]
	if !exists {
		return 0, fmt.Errorf("user_id claim is missing")
	}

	var userID int64
	switch v := userIDClaim.(type) {
	case float64:
		userID = int64(v)
	case int64:
		userID = v
	default:
		return 0, fmt.Errorf("user_id must be a number, got %T", v)
	}
	fmt.Println("token valid")
	return userID, nil
}

// Функция для извлечения userID из контекста
func GetUserIDFromContext(ctx context.Context) (int64, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return 0, errors.New("no metadata in context")
	}

	authHeader := md.Get("authorization")
	if len(authHeader) == 0 {
		return 0, errors.New("no authorization header")
	}

	tokenString := strings.TrimPrefix(authHeader[0], "Bearer ")
	if tokenString == authHeader[0] {
		return 0, errors.New("invalid authorization format")
	}

	return verifyToken(tokenString)
}
