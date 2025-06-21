package usecases

type UseCaseError struct {
	Code    int64
	Message string
}

func (e *UseCaseError) Error() string {
	return e.Message
}

var (
	ErrUserNotFound          = &UseCaseError{Code: -1001, Message: "user not found"}
	ErrPasswordNotMatch      = &UseCaseError{Code: -1002, Message: "password not match"}
	ErrEmailNotVerified      = &UseCaseError{Code: -1003, Message: "email not verified"}
	ErrEmailAlreadyExists    = &UseCaseError{Code: -1004, Message: "email already exists"}
	ErrUsernameAlreadyExists = &UseCaseError{Code: -1005, Message: "username already exists"}
	ErrEmailFormatInvalid    = &UseCaseError{Code: -1006, Message: "email format invalid"}
	ErrRoleInvalid           = &UseCaseError{Code: -1007, Message: "role invalid"}
	ErrMfaCodeRequired       = &UseCaseError{Code: -1008, Message: "mfa code required"}
	ErrMfaCodeNotMatch       = &UseCaseError{Code: -1009, Message: "mfa code not match"}
)
