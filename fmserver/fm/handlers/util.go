package handlers

type FMError struct {
	Code    int
	Message string
}

func GetError(code int, error string) FMError {
	return FMError{
		Code:    code,
		Message: error,
	}
}
