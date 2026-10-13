package jsonc

import (
	"testing"
)

func testNumber(t *testing.T, n string, shouldBeValid bool) {
	err := IsNumber(n)
	if shouldBeValid && err != nil {
		t.Fatalf("should be valid valid %v err: %v", n, err)
		return
	}

	if !shouldBeValid && err == nil {
		t.Fatalf("should not be valid %v err: %v", n, err)
		return
	}
}

func TestNumbers(t *testing.T) {

	type NumberTest struct {
		ShouldBeValid bool
		N             string
	}

	numberTests := []NumberTest{

		// --- Original Test Cases ---
		{ShouldBeValid: false, N: `09038749`},    // Leading zero on integer
		{ShouldBeValid: true, N: `9038749`},      // Standard integer
		{ShouldBeValid: true, N: `0.9038749`},    // Standard decimal
		{ShouldBeValid: false, N: `0..9038749`},  // Double decimal point
		{ShouldBeValid: true, N: `0.903e8749`},   // Standard scientific notation
		{ShouldBeValid: false, N: `0.9038ee749`}, // Double exponent sign
		{ShouldBeValid: false, N: `--9038749`},   // Double negative sign

		// --- Basic Signs & Zero Cases ---
		{ShouldBeValid: true, N: `0`},          // Single zero
		{ShouldBeValid: true, N: `-0`},         // Negative zero
		{ShouldBeValid: true, N: `-9038749`},   // Standard negative integer
		{ShouldBeValid: false, N: `+9038749`},  // Explicit plus (invalid in strict JSON, adjust if parsing Go)
		{ShouldBeValid: false, N: `-09038749`}, // Negative with leading zero

		// --- Decimals Edge Cases ---
		{ShouldBeValid: true, N: `-0.9038749`}, // Negative decimal
		{ShouldBeValid: false, N: `.9038749`},  // Missing leading zero before decimal
		{ShouldBeValid: false, N: `9038749.`},  // Trailing decimal point
		{ShouldBeValid: false, N: `-0.`},       // Negative trailing decimal

		// --- Scientific Notation (Exponent) Variations ---
		{ShouldBeValid: true, N: `9038749e10`},   // Integer with lowercase e
		{ShouldBeValid: true, N: `9038749E10`},   // Integer with uppercase E
		{ShouldBeValid: true, N: `9038749e+10`},  // Positive exponent with explicit plus
		{ShouldBeValid: true, N: `9038749e-10`},  // Negative exponent
		{ShouldBeValid: true, N: `-0.9038e-7`},   // Negative decimal with negative exponent
		{ShouldBeValid: false, N: `9038749e`},    // Missing exponent digits
		{ShouldBeValid: false, N: `9038749e+`},   // Missing exponent digits after plus
		{ShouldBeValid: false, N: `9038749e-`},   // Missing exponent digits after minus
		{ShouldBeValid: false, N: `9038749e1.5`}, // Non-integer exponent

		// --- Hexadecimal / Octal / Non-base10 (If applicable) ---
		// Note: Mark these as true if your parser supports Go/JS style hex, false if strictly JSON
		{ShouldBeValid: false, N: `0x1A2F`}, // Hexadecimal literal
		{ShouldBeValid: false, N: `0b1010`}, // Binary literal

		// --- Malformed Text & Empty Inputs ---
		{ShouldBeValid: false, N: ``},          // Empty string
		{ShouldBeValid: false, N: ` `},         // Whitespace only
		{ShouldBeValid: false, N: `9038a749`},  // Letter inside number
		{ShouldBeValid: false, N: `NaN`},       // NaN text
		{ShouldBeValid: false, N: `Infinity`},  // Infinity text
		{ShouldBeValid: false, N: `-Infinity`}, // Negative infinity text

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
			ShouldBeValid: false,
			N:             `--9038749`,
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
