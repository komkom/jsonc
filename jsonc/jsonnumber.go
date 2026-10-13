package jsonc

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

type NumberState int

const (
	NumberRoot NumberState = iota
	Digit
	DigitToDot
	DigitEnd
	Dot
	StartMinus
	FirstDigit
	Exp
)

const (
	NonZeroNumbers = "1,2,3,4,5,6,7,8,9"
	Numbers        = "0" + NonZeroNumbers
)

func IsNumber(value string) error {

	value = strings.TrimSpace(value)
	if value == "" {
		return errors.Errorf("wrong beginning")
	}
	last := value[len(value)-1]
	if last == '.' || last == 'e' || last == '+' || last == '-' {
		return fmt.Errorf("invalid ending")
	}

	s := StartMinus

	for _, v := range value {
	reloop:

		switch s {
		case StartMinus:

			if v == '-' {
				s = NumberRoot
				break
			}

			s = NumberRoot
			goto reloop

		case NumberRoot:

			if v == '0' {
				s = Dot
				break
			}
			if strings.Contains(NonZeroNumbers, string(v)) {
				s = DigitToDot
				break
			}

			return fmt.Errorf("numberRoot not a number")

		case Dot:
			if v == '.' {
				s = FirstDigit
				break
			}
			return fmt.Errorf("dot no dot")

		case Digit:
			if strings.Contains(Numbers, string(v)) {
				break
			}
			// has first number
			if v == 'E' || v == 'e' {
				s = Exp
				break
			}

			return fmt.Errorf("digit no digit")

		case FirstDigit:

			if strings.Contains(Numbers, string(v)) {
				s = Digit
				break
			}
			return fmt.Errorf("first digit no digit")

		case DigitToDot:
			if strings.Contains(Numbers, string(v)) {
				break
			}

			if v == '.' {
				s = FirstDigit
				break
			}

			if v == 'E' || v == 'e' {
				s = Exp
				break
			}

			return fmt.Errorf("digitToDot wrong")

		case Exp:
			if v == '-' || v == '+' {
				s = DigitEnd
				break
			}
			if strings.Contains(Numbers, string(v)) {
				s = DigitEnd
				break
			}

			return fmt.Errorf("exp wrong beginning")
		case DigitEnd:

			if strings.Contains(Numbers, string(v)) {
				break
			}
			return fmt.Errorf("exp digit end no number")
		}
	}

	return nil
}
