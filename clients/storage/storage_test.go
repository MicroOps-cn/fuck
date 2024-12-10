package storage

import (
	"io/fs"
	"testing"
)

func TestRawPermissionsToMode(t *testing.T) {

	tests := []struct {
		name string
		args string
		want fs.FileMode
	}{
		{name: "std-d755", args: "drwxr-xr-x", want: 0o755 | fs.ModeDir},
		{name: "std-755", args: "rwxr-xr-x", want: 0o755},
		{name: "std-514", args: "r-x--xr--", want: 0o514},
		{name: "std-777", args: "rwxrwxrwx", want: 0o777},
		{name: "std-d777", args: "drwxrwxrwx", want: 0o777 | fs.ModeDir},
		{name: "std-L777", args: "Lrwxrwxrwx", want: 0o777 | fs.ModeSymlink},
		{name: "std-000", args: "---------", want: 0o000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RawPermissionsToMode(tt.args); got != tt.want {
				t.Errorf("RawPermissionsToMode() = %v, want %v", got, tt.want)
			}
		})
	}
}
