package ghttp

type Error struct {
	StatusCode   int
	ResponseBody []byte
	cause        error
}

func NewError(statusCode int, body []byte, cause error) *Error {
	return &Error{
		StatusCode:   statusCode,
		ResponseBody: body,
		cause:        cause,
	}
}

func (e *Error) Cause() error {
	return e.cause
}

func (e *Error) Error() string {
	if e.ResponseBody != nil {
		return e.cause.Error() + ": " + string(e.ResponseBody)
	}

	return e.cause.Error()
}
