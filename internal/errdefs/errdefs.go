package errdefs

import "errors"

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrUserExists = errors.New("user already exists")
var ErrUserNotFound = errors.New("user not found")
