package random

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// ID random generator
var ID = make(chan string)

func init() {
	go func() {
		h := sha256.New()
		c := []byte(time.Now().String())
		for {
			_, _ = h.Write(c)
			ID <- fmt.Sprintf("%x", h.Sum(nil))
		}
	}()
}
