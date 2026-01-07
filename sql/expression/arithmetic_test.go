package expression

import (
	"testing"
)

func TestPlus(t *testing.T) {
	var testCases = []struct {
		name        string
		left, right float64
		expected    float64
	}{
		{"1 + 1", 1, 1, 2},
		{"-1 + 1", -1, 1, 0},
		{"0 + 0", 0, 0, 0},
		{"0.14159 + 3.0", 0.14159, 3.0, float64(0.14159) + float64(3)},
	}
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {

		})
	}
}
