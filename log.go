package signups

import "fmt"

type LogWriter struct{}

func (lw LogWriter) Write(p []byte) (int, error) {
	fmt.Println(string(p))
	fmt.Println("Wrote bytes:", len(p))
	return len(p), nil
}
