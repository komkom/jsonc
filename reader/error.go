package reader

import "fmt"

type errorf struct {
	error
	position int
}

func cerror(err error) *errorf {
	return &errorf{err, -1}
}

func errorF(formatter string, position int, args ...interface{}) *errorf {
	if position > 0 {
		return &errorf{fmt.Errorf("pos: %v "+formatter, position, args), position}
	}

	return &errorf{fmt.Errorf(formatter, args), position}
}

func errorFmt(formatter string, args ...interface{}) *errorf {
	return &errorf{fmt.Errorf(formatter, args), -1}
}
