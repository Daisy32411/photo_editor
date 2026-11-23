package models

import "errors"

var ErrTaskNotFound = errors.New("task not found")
var ErrUserNotFound = errors.New("user not found")
var ErrUserExists = errors.New("user already exists")
var ErrSessionNotFound = errors.New("session not found")
