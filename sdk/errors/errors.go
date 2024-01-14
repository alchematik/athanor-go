package sdk

func NewErrorNotFound() ErrorNotFound {
	return ErrorNotFound{}
}

type ErrorNotFound struct {
}

func (e ErrorNotFound) Error() string {
	return "resource not found"
}
