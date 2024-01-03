package w

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRing(t *testing.T) {
	r := NewRingQueue[int](10)
	for i := 0; i < 10; i++ {
		err := r.Put(i)
		if err != nil {
			require.NoError(t, err)
		}
	}
	require.Equal(t, r.Len(), 10)
	require.ErrorIs(t, r.Put(0), QueueFull)
	require.Equal(t, r.Len(), 10)
	for i := 0; i < 5; i++ {
		_, err := r.Get()
		require.NoError(t, err)
	}
	require.Equal(t, r.Len(), 5)
	for i := 0; i < 10; i++ {
		err := r.Put(i)
		if i >= 5 {
			require.ErrorIs(t, err, QueueFull)
		} else {
			require.NoError(t, err)
		}
	}
	require.Equal(t, r.List(), []int{5, 6, 7, 8, 9, 0, 1, 2, 3, 4})
	for i := 0; i < 3; i++ {
		_, err := r.Get()
		require.NoError(t, err)
	}
	for i := 0; i < 10; i++ {
		err := r.Put(i)
		if i >= 3 {
			require.ErrorIs(t, err, QueueFull)
		} else {
			require.NoError(t, err)
		}
	}
	require.Equal(t, r.List(), []int{8, 9, 0, 1, 2, 3, 4, 0, 1, 2})
	r.Remove(func(a int) bool {
		if a == 7 || a == 2 || a == 9 {
			return true
		}
		return false
	})
	require.Equal(t, r.List(), []int{8, 0, 1, 3, 4, 0, 1})
	r.Remove(func(a int) bool {
		if a == 8 || a == 0 {
			return true
		}
		return false
	})
	r.Do(func(a int) {
		require.Contains(t, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, a)
	})
	require.Equal(t, r.List(), []int{1, 3, 4, 1})
	for i := 0; i < 4; i++ {
		_, err := r.Get()
		require.NoError(t, err)
	}
	require.Equal(t, r.Len(), 0)
	for i := 0; i < 4; i++ {
		_, err := r.Get()
		require.ErrorIs(t, err, QueueNull)
	}

	require.Equal(t, r.Len(), 0)
	for i := 0; i < 7; i++ {
		err := r.Put(i)
		if err != nil {
			require.NoError(t, err)
		}
	}
	require.Equal(t, r.Len(), 7)
	for i := 0; i < 7; i++ {
		_, err := r.Get()
		if err != nil {
			require.NoError(t, err)
		}
	}
	require.Equal(t, r.Len(), 0)
}
