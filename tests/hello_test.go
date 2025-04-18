package tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelloWorld(t *testing.T) {
	helper := NewHelper()
	err := helper.Db.Snapshot()
	require.NoError(t, err)
	defer func() {
		err := helper.Db.Rollback()
		require.NoError(t, err)
	}()

	msg := "hello world"
	require.EqualValues(t, "hello world", msg)
}
