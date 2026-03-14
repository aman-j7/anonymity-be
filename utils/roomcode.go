package utils

import "math/rand"

const (
	codeChars  = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeLength = 5
)

func GenerateRoomCode(exists func(string) bool) string {
	for {
		code := make([]byte, codeLength)
		for i := range code {
			code[i] = codeChars[rand.Intn(len(codeChars))]
		}
		s := string(code)
		if !exists(s) {
			return s
		}
	}
}
