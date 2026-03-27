package apperr

type NotFound struct {
	Description string
	Err         error
}

func (e *NotFound) Error() string { return e.Description }
func (e *NotFound) Unwrap() error { return e.Err }

type AlreadyExists struct {
	Description string
	Err         error
}

func (e *AlreadyExists) Error() string { return e.Description }
func (e *AlreadyExists) Unwrap() error { return e.Err }

type InvalidInput struct {
	Description string
	Err         error
}

func (e *InvalidInput) Error() string { return e.Description }
func (e *InvalidInput) Unwrap() error { return e.Err }

type Forbidden struct {
	Description string
	Err         error
}

func (e *Forbidden) Error() string { return e.Description }
func (e *Forbidden) Unwrap() error { return e.Err }

type Conflict struct {
	Description string
	Err         error
}

func (e *Conflict) Error() string { return e.Description }
func (e *Conflict) Unwrap() error { return e.Err }
