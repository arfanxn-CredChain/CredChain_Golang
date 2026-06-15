package seeder

import "strings"

// SanitizePhone normalizes a phone number to strict E.164 format.
//
// Transformations:
//  1. Strip all whitespace characters (spaces, tabs, etc.)
//  2. If the result starts with "0", replace the leading "0" with "+62"
//  3. If the result does not start with "+", prefix it with "+62"
//
// Returns an empty string if the input is empty or only whitespace.
// The output is guaranteed to pass the strictE164Rule regex ^\+[1-9]\d{6,14}$
// provided the input contained a valid phone number body.
func SanitizePhone(phone string) string {
	phone = strings.Map(func(r rune) rune {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			return -1
		}
		return r
	}, phone)

	if phone == "" {
		return ""
	}

	if phone[0] == '0' {
		phone = "+62" + phone[1:]
	} else if phone[0] != '+' {
		phone = "+62" + phone
	}

	return phone
}
