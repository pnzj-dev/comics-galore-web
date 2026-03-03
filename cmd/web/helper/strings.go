package helper

import (
	"crypto/rand"
	"fmt"
)

func RandomID() string {
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}
