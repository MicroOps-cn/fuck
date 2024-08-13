package safe

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"golang.org/x/crypto/sha3"
	"hash"
	"reflect"
	"testing"
)

func TestHashSum(t *testing.T) {
	type args struct {
		hashFunc  func() hash.Hash
		data      []byte
		hexLength int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{name: "md5", args: args{hashFunc: md5.New, data: []byte("test"), hexLength: 32}, want: "098f6bcd4621d373cade4e832627b4f6"},
		{name: "md5-16", args: args{hashFunc: md5.New, data: []byte("test"), hexLength: 16}, want: "098f6bcd4621d373"},
		{name: "sha1", args: args{hashFunc: sha1.New, data: []byte("test"), hexLength: 32}, want: "a94a8fe5ccb19ba61c4c0873d391e987"},
		{name: "sha256", args: args{hashFunc: sha256.New, data: []byte("test"), hexLength: 100}, want: "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"},
		{name: "sha384", args: args{hashFunc: sha512.New384, data: []byte("test"), hexLength: 100}, want: "768412320f7b0aa5812fce428dc4706b3cae50e02a64caa16a782249bfe8efc4b7ef1ccb126255d196047dfedf17a0a9"},
		{name: "sha512", args: args{hashFunc: sha512.New, data: []byte("test"), hexLength: 100}, want: "ee26b0dd4af7e749aa1a8ee3c10ae9923f618980772e473f8819a5d4940e0db27ac185f8a0e1d5f84f88bc887fd67b143732"},
		{name: "sha3-384", args: args{hashFunc: sha3.New384, data: []byte("test"), hexLength: 100}, want: "e516dabb23b6e30026863543282780a3ae0dccf05551cf0295178d7ff0f1b41eecb9db3ff219007c4e097260d58621bd"},
		{name: "sha3-512", args: args{hashFunc: sha3.New512, data: []byte("test"), hexLength: 100}, want: "9ece086e9bac491fac5c1d1046ca11d737b92a2b2ebd93f005d7b710110c0a678288166e7fbe796883a4f2e9b3ca9f484f52"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewHash(tt.args.hashFunc, tt.args.data).HexString(tt.args.hexLength); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HashSum() = %v, want %v", got, tt.want)
			}
		})
	}
}
