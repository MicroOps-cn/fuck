package safe

import (
	"bytes"
	"encoding/json"
	"io"
	"reflect"
	"testing"

	"github.com/gogo/protobuf/jsonpb"
	jsonpb2 "github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
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
				t.Errorf("MarshalJSON() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncryptedString_MarshalJSONPB(t *testing.T) {
	type args struct {
		marshaler *jsonpb.Marshaler
	}
	tests := []struct {
		name    string
		e       String
		args    args
		want    []byte
		wantErr bool
	}{{
		name: "simple", e: String{Value: "hello"}, want: []byte(`"hello"`), wantErr: false,
		args: args{marshaler: &jsonpb.Marshaler{}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.args.marshaler.MarshalToString(&tt.e)
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
	Unmarshal(r io.Reader, m proto.Message) error
}
type golangUnmarshaller interface {
	Unmarshal(r io.Reader, m proto.Message) error
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
		args: args{unmarshaler: &jsonpb2.Unmarshaler{}, bytes: []byte(`"hello"`)},
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
