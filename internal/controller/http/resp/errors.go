package resp

type APIError struct {
	Code    int64
	Message string
}

func (e *APIError) Error() string {
	return e.Message
}

var (
	ErrTypeWrong          = &APIError{Code: -101, Message: "type wrong"}
	ErrIDRequired         = &APIError{Code: -102, Message: "id required"}
	ErrGetDiskUsageFailed = &APIError{Code: -103, Message: "get disk usage failed"}
	ErrNameRequired       = &APIError{Code: -104, Message: "name required"}
	ErrNotFound           = &APIError{Code: -105, Message: "not found"}
	ErrEmailFormatInvalid = &APIError{Code: -106, Message: "email format invalid"}
	ErrEmailRequired      = &APIError{Code: -107, Message: "email required"}
	ErrCannotDeleteSelf   = &APIError{Code: -108, Message: "cannot delete self"}
)
