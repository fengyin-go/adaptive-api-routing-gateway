package model

// ValidationError 表示字段校验失败。
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return e.Field + ": " + e.Message
	}
	return e.Message
}

func NewValidationError(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}

func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}
