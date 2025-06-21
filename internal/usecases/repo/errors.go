package repo

type Error struct {
	Code    int64
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

var errInsertFail = &Error{Code: -31, Message: "insert fail"}
