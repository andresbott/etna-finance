package csvimport

type ErrValidation string

func (v ErrValidation) Error() string {
	return string(v)
}
