package workerpool_test

import (
	"testing"

	"github.com/sean-/patterns/workerpool"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	wp := workerpool.New(
		workerpool.Config{},
		workerpool.Factories{},
		workerpool.Handlers{},
	)
	require.NotNil(t, wp)
}
