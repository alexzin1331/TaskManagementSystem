package models

import (
	"fmt"
	"time"
)

type User struct {
	ID       int64
	Email    string
	PassHash []byte
}

type Task struct {
	ID          int64
	UserID      int64
	Title       string
	Description string
	DueDate     *time.Time // Pointer to time.Time to handle NULL values from DB
	Status      int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type TaskStatus int32

const (
	TASK_STATUS_UNSPECIFIED TaskStatus = 0
	TASK_STATUS_OPEN        TaskStatus = 1
	TASK_STATUS_IN_PROGRESS TaskStatus = 2
	TASK_STATUS_COMPLETED   TaskStatus = 3
	TASK_STATUS_PENDING     TaskStatus = 4
	TASK_STATUS_CANCELLED   TaskStatus = 5
)

// String returns the string representation of the TaskStatus.
func (ts TaskStatus) String() string {
	switch ts {
	case TASK_STATUS_UNSPECIFIED:
		return "UNSPECIFIED"
	case TASK_STATUS_OPEN:
		return "OPEN"
	case TASK_STATUS_IN_PROGRESS:
		return "IN_PROGRESS"
	case TASK_STATUS_COMPLETED:
		return "COMPLETED"
	case TASK_STATUS_PENDING:
		return "PENDING"
	case TASK_STATUS_CANCELLED:
		return "CANCELLED"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int32(ts))
	}
}

const (
	Secret = "secret"
)
