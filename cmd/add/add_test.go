package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAdd(t *testing.T) {
	testCases := []struct {
		Name     string
		A        int
		B        int
		Expected int
	}{
		{
			Name:     "positive numbers",
			A:        5,
			B:        10,
			Expected: 15,
		},
		{
			Name:     "negative numbers",
			A:        -5,
			B:        -10,
			Expected: -15,
		},
		{
			Name:     "mixed positive and negative",
			A:        10,
			B:        -5,
			Expected: 5,
		},
		{
			Name:     "zero values",
			A:        0,
			B:        0,
			Expected: 0,
		},
		{
			Name:     "one zero value",
			A:        5,
			B:        0,
			Expected: 5,
		},
		{
			Name:     "large numbers",
			A:        1000000,
			B:        2000000,
			Expected: 3000000,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			result := Add(testCase.A, testCase.B)
			require.Equal(t, testCase.Expected, result)
		})
	}
}

func TestAddMultiple(t *testing.T) {
	testCases := []struct {
		Name     string
		Numbers  []int
		Expected int
	}{
		{
			Name:     "multiple positive numbers",
			Numbers:  []int{1, 2, 3, 4, 5},
			Expected: 15,
		},
		{
			Name:     "multiple negative numbers",
			Numbers:  []int{-1, -2, -3},
			Expected: -6,
		},
		{
			Name:     "mixed numbers",
			Numbers:  []int{10, -5, 3, -2},
			Expected: 6,
		},
		{
			Name:     "single number",
			Numbers:  []int{42},
			Expected: 42,
		},
		{
			Name:     "empty slice",
			Numbers:  []int{},
			Expected: 0,
		},
		{
			Name:     "zeros",
			Numbers:  []int{0, 0, 0},
			Expected: 0,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			result := AddMultiple(testCase.Numbers...)
			require.Equal(t, testCase.Expected, result)
		})
	}
}
