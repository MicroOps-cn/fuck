package conv

import (
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/require"
)

type testStructE struct {
	raw   string
	value int
}

// StringToTimeHookFunc returns a DecodeHookFunc that converts
// strings to time.Time.
func testStructEHookFunc(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if t != reflect.TypeOf(testStructE{}) {
		return data, nil
	}
	if f.Kind() != reflect.String {
		return data, nil
	}
	i, _ := strconv.Atoi(data.(string))
	return testStructE{raw: data.(string), value: i}, nil
}

var _ mapstructure.DecodeHookFuncType = testStructEHookFunc

type testStruct[eT any] struct {
	A []string
	B int
	C float32
	D string
	E eT
	F time.Time
}

func TestDecodeQuery(t *testing.T) {
	type args struct {
		query   string
		dst     interface{}
		wantDst interface{}
		config  *mapstructure.DecoderConfig
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{{
		name: "query",
		args: args{
			query: "a=abc&b=2&c=3.0&d=4&a=6&E=102&F=1703582405.445311",
			dst:   testStruct[testStructE]{},
			wantDst: testStruct[testStructE]{
				A: []string{"abc", "6"}, B: 2, C: 3.0, D: "4", E: testStructE{raw: `102`, value: 102},
				F: time.Date(2023, time.December, 26, 17, 20, 5, 445000000, time.Local),
			},
		},
		wantErr: false,
	}, {
		name: "query point",
		args: args{
			query: "a=abc&b=2&c=3.0&d=4&a=6&E=102&F=1703582405.445311",
			dst:   testStruct[*testStructE]{},
			wantDst: testStruct[*testStructE]{
				A: []string{"abc", "6"}, B: 2, C: 3.0, D: "4", E: &testStructE{raw: `102`, value: 102},
				F: time.Date(2023, time.December, 26, 17, 20, 5, 445000000, time.Local),
			},
		},
		wantErr: false,
	}, {
		name: "query mill",
		args: args{
			query: "a=abc&b=2&c=3.0&d=4&a=6&E=102&F=1703582405445.311",
			dst:   testStruct[*testStructE]{},
			wantDst: testStruct[*testStructE]{
				A: []string{"abc", "6"}, B: 2, C: 3.0, D: "4", E: &testStructE{raw: `102`, value: 102},
				F: time.Date(2023, time.December, 26, 17, 20, 5, 445000000, time.Local),
			},
		},
		wantErr: false,
	}, {
		name: "query error",
		args: args{
			query: "a=abc&b=2&c=3.0&d=4&a=6&E=102&F=1703582405.445311",
			dst:   testStruct[*struct{}]{},
			wantDst: testStruct[*struct{}]{
				A: []string{"abc", "6"}, B: 2, C: 3.0, D: "4", E: &struct{}{},
				F: time.Date(2023, time.December, 26, 17, 20, 5, 445311069, time.Local),
			},
		},
		wantErr: true,
	}, {
		name: "default",
		args: args{
			query: `a=abc&c=3.0&d=4&a=6&E={"C":1}&F=1703582405.445311`,
			dst: testStruct[testStruct[any]]{
				A: []string{"1x"}, B: 99, C: 3.0, D: "4", E: testStruct[any]{B: 3, A: []string{"x"}},
			},
			wantDst: testStruct[testStruct[any]]{
				A: []string{"abc", "6"}, B: 99, C: 3.0, D: "4", E: testStruct[any]{
					B: 3, A: []string{"x"}, C: 1,
				},
				F: time.Date(2023, time.December, 26, 17, 20, 5, 445000000, time.Local),
			},
		},
		wantErr: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := DecodeQuery(tt.args.query, &tt.args.dst, tt.args.config, testStructEHookFunc); (err != nil) != tt.wantErr {
				t.Errorf("DecodeQuery() error = %v, wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr {
				require.Equal(t, tt.args.wantDst, tt.args.dst)
			}
		})
	}
}
