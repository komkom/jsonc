package jsonc

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
		{
			ShouldBeValid: false,
			N:             `09038749`,
		},
		{
			ShouldBeValid: true,
			N:             `9038749`,
		},
		{
			ShouldBeValid: true,
			N:             `0.9038749`,
		},
		{
			ShouldBeValid: false,
			N:             `0..9038749`,
		},
		{
			ShouldBeValid: true,
			N:             `0.903e8749`,
		},
		{
			ShouldBeValid: false,
			N:             `0.9038ee749`,
		},
		{
			ShouldBeValid: true,
			N:             `-9038749`,
		},
		{
			ShouldBeValid: true,
			N:             `-0.9038749`,
		},
		{
			ShouldBeValid: false,
			N:             `012435.9038749`,
		},
		{
			ShouldBeValid: true,
			N:             `12435.9038749`,
		},
		{
			ShouldBeValid: false,
			N:             `0e`,
		},
		{
			ShouldBeValid: true,
			N:             `0`,
		},
	}

	for _, nt := range numberTests {
		testNumber(t, nt.N, nt.ShouldBeValid)
	}
}
