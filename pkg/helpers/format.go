package helpers

import "strconv"

// FormatThousands renders a non-negative count with thousands separators, for
// truncation markers that report how much data was left behind.
func FormatThousands(n int) string {
	digits := strconv.Itoa(n)
	if len(digits) <= 3 {
		return digits
	}
	var formatted []rune
	for i, digit := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			formatted = append(formatted, ',')
		}
		formatted = append(formatted, digit)
	}
	return string(formatted)
}
