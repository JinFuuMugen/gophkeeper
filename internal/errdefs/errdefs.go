package errdefs

import "errors"

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrUserExists          = errors.New("user already exists")
	ErrUserNotFound        = errors.New("user not found")
	ErrBadConfigValue      = errors.New("cannot validate config value")
	ErrNoPassword          = errors.New("password required: use GK_PASSWORD")
	ErrNoCredentials       = errors.New("login and password required")
	ErrTitleRequired       = errors.New("title required")
	ErrTextRequired        = errors.New("text required")
	ErrPathRequired        = errors.New("path required")
	ErrIDRequired          = errors.New("id required")
	ErrInvalidID           = errors.New("invalid id")
	ErrItemNotFound        = errors.New("item not found in local cache; run: gophkeeper sync")
	ErrItemDeleted         = errors.New("item is deleted")
	ErrMissingPayload      = errors.New("missing encrypted payload in cache")
	ErrEmptyKDFSalt        = errors.New("server returned empty kdf_salt_b64")
	ErrCardFieldsRequired  = errors.New("title, number, exp, cvc required")
	ErrLoginFieldsRequired = errors.New("title, login, password required")
	ErrMasterPassRequired  = errors.New("master password required: use --master-pass, GK_MASTER_PASS env")
)
