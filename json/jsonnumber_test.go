package json

import "testing"

func testNumber(t *testing.T, n string, shouldBeValid bool) {
	b := IsNumber(n)
	if shouldBeValid && !b {
		t.Fatalf("should be valid valid %v", n)
		return
	}

	if !shouldBeValid && b {
		t.Fatalf("should not be valid %v", n)
		return
	}
}

func TestNumbers(t *testing.T) {

	type NumberTest struct {
		ShouldBeValid bool
		N             string
	}

	numberTests := []NumberTest{
		NumberTest{
			ShouldBeValid: false,
			N:             `09038749`,
		},
		NumberTest{
			ShouldBeValid: true,
			N:             `9038749`,
		},
		NumberTest{
			ShouldBeValid: true,
			N:             `0.9038749`,
		},
		NumberTest{
			ShouldBeValid: false,
			N:             `0..9038749`,
		},
		NumberTest{
			ShouldBeValid: true,
			N:             `0.903e8749`,
		},
		NumberTest{
			ShouldBeValid: false,
			N:             `0.9038ee749`,
		},
		NumberTest{
			ShouldBeValid: true,
			N:             `-9038749`,
		},
		NumberTest{
			ShouldBeValid: true,
			N:             `-0.9038749`,
		},
		NumberTest{
			ShouldBeValid: false,
			N:             `012435.9038749`,
		},
		NumberTest{
			ShouldBeValid: true,
			N:             `12435.9038749`,
		},
	}

	for _, nt := range numberTests {
		testNumber(t, nt.N, nt.ShouldBeValid)
	}
}
