package utils

import "math/rand"

const (
	codeChars     = "!@#$%^&*ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	specialChars  = "!@#$%^&*"
	codeLength    = 6
)

func GenerateRoomCode(exists func(string) bool) string {
	for {
		code := make([]byte, codeLength)
		hasSpecial := false

		for i := range code {
			c := codeChars[rand.Intn(len(codeChars))]
			code[i] = c

			if containsSpecial(c) {
				hasSpecial = true
			}
		}

		if !hasSpecial {
			continue
		}

		s := string(code)
		if !exists(s) {
			return s
		}
	}
}

func containsSpecial(c byte) bool {
	for i := 0; i < len(specialChars); i++ {
		if c == specialChars[i] {
			return true
		}
	}
	return false
}
