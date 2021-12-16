package amigo

import (
	"fmt"
	"math/rand"
	"time"
)

var rnd *rand.Rand

func init() {
	rnd = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func generateActionID(length int) string {
	l := length/2 + length%2

	b := make([]byte, l)

	_, err := rnd.Read(b[:])
	if err != nil {
		panic(err)
	}

	return fmt.Sprintf("%x", b)[:length]
}
