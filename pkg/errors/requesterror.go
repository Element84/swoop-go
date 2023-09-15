package errors

type RequestError struct {
	Retryable bool
	Err       error
}

func NewRequestError(err error, retryable bool) *RequestError {
	return &RequestError{
		Err:       err,
		Retryable: retryable,
	}
}

func (re *RequestError) Error() string {
	return re.Err.Error()
}
