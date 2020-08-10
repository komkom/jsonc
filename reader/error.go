package reader

import "fmt"

type Error struct {
	err      string
	position int
}

func (e Error) Error() string {
	return e.err
}

func (e Error) Position() int {
	return e.position
}

func Errorf(formatter string, position int, args ...interface{}) Error {

	if position > 0 {

		jargs := []interface{}{position}
		jargs = append(jargs, args...)

		return Error{err: fmt.Sprintf("pos: %v "+formatter, jargs...), position: position}
	}

	return Error{err: fmt.Sprintf(formatter, args), position: position}
}
