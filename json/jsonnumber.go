package json

import "strings"

type State int

const (
	Root State = iota
	ZeroStart
	Digit
	PlusMinus
)

const (
	NonZeroNumbers = "1,2,3,4,5,6,7,8,9"
	Numbers        = "0" + NonZeroNumbers
)

func IsNumber(value string) bool {

	if value == "" {
		return false
	}

	s := Root

	var hasDot bool
	var lastRune rune
	for _, v := range value {

		lastRune = v

		switch s {
		case Root:

			if v == '-' {
				break
			}

			if v == '0' {
				s = ZeroStart
				break
			}

			if strings.ContainsRune(NonZeroNumbers, v) {
				s = Digit
				break
			}

			return false

		case ZeroStart:

			if v == '.' {
				hasDot = true
				s = Digit
				break
			}

			if v == 'e' || v == 'E' {
				s = PlusMinus
				break
			}

			return false

		case Digit:

			if v == '.' {
				if hasDot {
					return false
				}
				hasDot = true
				s = Digit
				break
			}

			if v == 'e' || v == 'E' {
				s = PlusMinus
				break
			}

			if strings.ContainsRune(Numbers, v) {
				break
			}

			return false

		case PlusMinus:

			if strings.ContainsRune("+-", v) {
				s = Digit
				break
			}

			if strings.ContainsRune(Numbers, v) {
				s = Digit
				break
			}

			return false
		}
	}

	return (s == Digit && strings.ContainsRune(Numbers, lastRune)) ||
		s == PlusMinus ||
		s == ZeroStart
}
