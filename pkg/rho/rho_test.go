package rho

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Map(t *testing.T) {
	arr := []int{1, 2, 3, 4}
	result := Map(arr, func(v int, _ int) int { return v * 2 })

	require.Equal(t, []int{2, 4, 6, 8}, result)
}
