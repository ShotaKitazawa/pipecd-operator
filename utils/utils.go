package utils

import (
	"math"
	"math/rand"
	"time"
)

const (
	letters    = "abcdefghijklmnopqrstuvwxyz"
	numOfRetry = 4
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func RandString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[int(rand.Int63()%int64(len(letters)))]
	}
	return string(b)
}

func ExpBackoff(f func() error) (err error) {
	for i := 0; i < numOfRetry; i++ {
		if err = f(); err == nil {
			return nil
		}
		time.Sleep(time.Second*time.Duration(math.Pow(2, float64(i))) + time.Duration(rand.Intn(100)*100))
	}
	return err
}
