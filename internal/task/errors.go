package task

import "errors"

var (
	// repository error
	errNotFound   = errors.New("task not found")
	errUnique     = errors.New("error unique")
	errForeignKey = errors.New("error foreign key")

	// service error
	errInvalidTask  = errors.New("invalid task")
	errTaskNotFound = errors.New("task not found")
	errUserNotFound = errors.New("user not found")
)
