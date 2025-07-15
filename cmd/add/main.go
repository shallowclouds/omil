package main

import (
	"fmt"
	"os"
	"strconv"
)

func Add(a, b int) int {
	return a + b
}

func AddMultiple(numbers ...int) int {
	sum := 0
	for _, num := range numbers {
		sum += num
	}
	return sum
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <number1> <number2> [number3] ...")
		fmt.Println("Example: go run main.go 5 10 15")
		return
	}

	var numbers []int
	for i := 1; i < len(os.Args); i++ {
		num, err := strconv.Atoi(os.Args[i])
		if err != nil {
			fmt.Printf("Error: '%s' is not a valid integer\n", os.Args[i])
			return
		}
		numbers = append(numbers, num)
	}

	if len(numbers) == 2 {
		result := Add(numbers[0], numbers[1])
		fmt.Printf("Result: %d + %d = %d\n", numbers[0], numbers[1], result)
	} else {
		result := AddMultiple(numbers...)
		fmt.Printf("Result: %v = %d\n", numbers, result)
	}
}
