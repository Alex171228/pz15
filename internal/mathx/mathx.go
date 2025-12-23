package mathx

import "errors"

func Sum(a, b int) int { return a + b }

func Divide(a, b int) (int, error) {
	if b == 0 {
		return 0, errors.New("divide by zero")
	}
	return a / b, nil
}

// Fib is a slow recursive Fibonacci implementation (intentionally) for benchmarking.
func Fib(n int) int {
	if n <= 1 {
		return n
	}
	return Fib(n-1) + Fib(n-2)
}

// FibFast is an optimized iterative Fibonacci implementation.
func FibFast(n int) int {
	if n <= 1 {
		return n
	}
	a, b := 0, 1
	for i := 2; i <= n; i++ {
		a, b = b, a+b
	}
	return b
}
