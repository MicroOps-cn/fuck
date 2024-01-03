package safe

import (
	"bytes"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"os"
	"reflect"
	"testing"

	gogojsonpb "github.com/gogo/protobuf/jsonpb"
	gogoproto "github.com/gogo/protobuf/proto"
	golangjsonpb "github.com/golang/protobuf/jsonpb"
	golangproto "github.com/golang/protobuf/proto"
)

func TestEncryptedString_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		e       String
		want    []byte
		wantErr bool
	}{{
		name: "simple", e: String{Value: "hello"}, want: []byte(`"hello"`), wantErr: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.e)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", string(got), string(tt.want))
			}
		})
	}
}

type gogoMarshaller interface {
	MarshalToString(m gogoproto.Message) (string, error)
}
type golangMarshaller interface {
	MarshalToString(m golangproto.Message) (string, error)
}

func TestEncryptedString_MarshalJSONPB(t *testing.T) {
	type args struct {
		marshaler any
	}
	tests := []struct {
		name    string
		e       String
		args    args
		want    []byte
		wantErr bool
	}{{
		name: "simple", e: String{Value: "hello"}, want: []byte(`"hello"`), wantErr: false,
		args: args{marshaler: &gogojsonpb.Marshaler{}},
	}, {
		name: "simple", e: String{Value: "hello"}, want: []byte(`"hello"`), wantErr: false,
		args: args{marshaler: &golangjsonpb.Marshaler{}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got string
			var err error
			switch unmarshaler := tt.args.marshaler.(type) {
			case gogoMarshaller:
				got, err = unmarshaler.MarshalToString(&tt.e)
			case golangMarshaller:
				got, err = unmarshaler.MarshalToString(&tt.e)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSONPB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual([]byte(got), tt.want) {
				t.Errorf("MarshalJSONPB() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncryptedString_UnmarshalJSON(t *testing.T) {
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name    string
		e       String
		args    args
		wantErr bool
	}{{
		name: "simple", e: String{Value: "hello"}, wantErr: false,
		args: args{bytes: []byte(`"hello"`)},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e String
			if err := json.Unmarshal(tt.args.bytes, &e); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(e, tt.e) {
				t.Errorf("MarshalJSONPB() got = %v, want %v", e, tt.e)
			}
		})
	}
}

type gogoUnmarshaller interface {
	Unmarshal(r io.Reader, m gogoproto.Message) error
}
type golangUnmarshaller interface {
	Unmarshal(r io.Reader, m golangproto.Message) error
}

func TestEncryptedString_UnmarshalJSONPB(t *testing.T) {
	type args struct {
		unmarshaler any
		bytes       []byte
	}
	tests := []struct {
		name    string
		e       String
		args    args
		wantErr bool
	}{{
		name: "simple", e: String{Value: "hello"}, wantErr: false,
		args: args{unmarshaler: &golangjsonpb.Unmarshaler{}, bytes: []byte(`"hello"`)},
	}, {
		name: "simple2", e: String{Value: "hello"}, wantErr: false,
		args: args{unmarshaler: &gogojsonpb.Unmarshaler{}, bytes: []byte(`"hello"`)},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var e String
			var err error
			buf := bytes.NewReader(tt.args.bytes)
			switch unmarshaler := tt.args.unmarshaler.(type) {
			case gogoUnmarshaller:
				err = unmarshaler.Unmarshal(buf, &e)
			case golangUnmarshaller:
				err = unmarshaler.Unmarshal(buf, &e)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSONPB() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(e, tt.e) {
				t.Errorf("MarshalJSONPB() got = %v, want %v", e, tt.e)
			}
		})
	}
}

func TestNewEncryptedString(t *testing.T) {
	type args struct {
		unSafeString string
		secret       string
		o            *EncryptOptions
	}
	tests := []struct {
		name string
		args args
	}{{
		name: "unsafe",
		args: args{unSafeString: "123456"},
	}, {
		name: "simple",
		args: args{unSafeString: "123456", secret: "$L9+W9M!jbGMPjKln7Rn6Ge."},
	}, {
		name: "simple aes",
		args: args{unSafeString: "123456", o: NewEncryptOptions(WithAlgorithm(AlgorithmAES)), secret: "$L9+W9M!jbGMPjKln7Rn6Ge."},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plain := tt.args.unSafeString
			if len(tt.args.secret) != 0 {
				var err error
				plain, err = Encrypt([]byte(tt.args.unSafeString), tt.args.secret, tt.args.o)
				require.NoError(t, err)
			}
			got := NewEncryptedString(plain, tt.args.secret)
			unSafeString, err := got.UnsafeString()
			require.NoError(t, err)
			if len(tt.args.secret) > 0 {
				require.NotEqual(t, got.Value, tt.args.unSafeString)
			}

			if !reflect.DeepEqual(unSafeString, tt.args.unSafeString) {
				t.Errorf("NewEncryptedString().UnsafeString()= %v, want %v", unSafeString, tt.args.unSafeString)
			}

			err = os.Setenv(SecretEnvName, tt.args.secret)
			require.NoError(t, err)
			got.secret = ""
			unSafeString, err = got.UnsafeString()
			require.NoError(t, err)
			if len(tt.args.secret) > 0 {
				require.NotEqual(t, got.Value, tt.args.unSafeString)
			}

			if !reflect.DeepEqual(unSafeString, tt.args.unSafeString) {
				t.Errorf("NewEncryptedString().UnsafeString()= %v, want %v", unSafeString, tt.args.unSafeString)
			}
			err = os.Unsetenv(SecretEnvName)
			require.NoError(t, err)
		})
	}
}
