package api

import "fmt"

func NewUnusedFunction() {
	fmt.Print("will linter pick up on this unused function")
}
