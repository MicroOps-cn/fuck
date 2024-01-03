package conv

import (
	"encoding"
	"encoding/json"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/mitchellh/mapstructure"

	"github.com/MicroOps-cn/fuck/capacity"
)

// DecodeQuery takes an input query string and uses reflection to translate it to
// the dst structure. dst must be a pointer to a map or struct.
func DecodeQuery(query string, dst interface{}, config *mapstructure.DecoderConfig, fs ...mapstructure.DecodeHookFunc) error {
	u, err := url.ParseQuery(query)
	if err != nil {
		return err
	}
	return DecodeURLValues(u, dst, config, fs...)
}

var defaultDecodeHookFuncs = []mapstructure.DecodeHookFunc{
	StringToTimeHookFunc,
	BinaryUnmarshalerHookFunc(),
	mapstructure.TextUnmarshallerHookFunc(),
	mapstructure.StringToIPHookFunc(),
	mapstructure.StringToIPNetHookFunc(),
	mapstructure.StringToTimeDurationHookFunc(),
	capacity.StringToCapacityHookFunc(),
}

func DefaultDecodeHookFuncs() []mapstructure.DecodeHookFunc {
	funcs := make([]mapstructure.DecodeHookFunc, len(defaultDecodeHookFuncs))
	copy(funcs, defaultDecodeHookFuncs)
	return funcs
}

// DecodeURLValues takes an input url.Values and uses reflection to translate it to
// the dst structure. dst must be a pointer to a map or struct.
func DecodeURLValues(u url.Values, dst interface{}, config *mapstructure.DecoderConfig, fs ...mapstructure.DecodeHookFunc) error {
	if config == nil {
		config = &mapstructure.DecoderConfig{
			DecodeHook: mapstructure.ComposeDecodeHookFunc(DefaultDecodeHookFuncs()...),
		}
	}
	config.Result = dst
	fs = append(fs, config.DecodeHook, StringToBaseTypeHookFunc)
	config.DecodeHook = URLValuesHookFunc(mapstructure.ComposeDecodeHookFunc(fs...))
	decoder, err := mapstructure.NewDecoder(config)
	if err != nil {
		return err
	}
	return decoder.Decode(u)
}

// StringToTimeHookFunc returns a DecodeHookFunc that converts
// strings to time.Time.
func StringToTimeHookFunc(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	if f.Kind() != reflect.String {
		return data, nil
	}
	if t != reflect.TypeOf(time.Time{}) {
		return data, nil
	}

	if ts, err := strconv.ParseInt(data.(string), 10, 64); err == nil {
		if ts > 1e12 {
			return time.UnixMilli(ts), nil
		}
		return time.Unix(ts, 0), nil
	}
	if ts, err := strconv.ParseFloat(data.(string), 64); err == nil {
		if ts > 1e12 {
			return time.UnixMilli(int64(ts)), nil
		}
		return time.Unix(int64(ts), int64((ts-float64(int64(ts)))*1e3)%1e3*1e6), nil
	}
	var tm time.Time
	var err error
	for _, timeFormat := range timeFormats {
		tm, err = time.Parse(timeFormat, data.(string))
		if err == nil {
			break
		}
	}
	return tm, err
}

func StringToBaseTypeHookFunc(fv reflect.Value, tv reflect.Value) (interface{}, error) {
	data := fv.Interface()
	if fv.Kind() != reflect.String {
		return data, nil
	}
	if fv.Kind() == reflect.String {
		switch tv.Kind() {
		case reflect.Int,
			reflect.Int8,
			reflect.Int16,
			reflect.Int32,
			reflect.Int64,
			reflect.Uint,
			reflect.Uint8,
			reflect.Uint16,
			reflect.Uint32,
			reflect.Uint64:
			return strconv.ParseInt(data.(string), 10, 64)
		case reflect.Float64,
			reflect.Float32:
			return strconv.ParseFloat(data.(string), 64)
		case reflect.Bool:
			return strconv.ParseBool(data.(string))
		case reflect.Struct, reflect.Map:
			v := tv.Interface()
			err := json.Unmarshal([]byte(data.(string)), &v)
			if err != nil {
				return nil, err
			}
			return v, nil
		}
	}
	return data, nil
}

func URLValuesHookFunc(fc mapstructure.DecodeHookFunc) mapstructure.DecodeHookFunc {
	return func(fv reflect.Value, tv reflect.Value) (interface{}, error) {
		data, err := mapstructure.DecodeHookExec(fc, fv, tv)
		if err != nil {
			return nil, err
		}
		fv = reflect.ValueOf(data)
		if fv.Kind() == reflect.Slice && tv.Kind() != reflect.Slice {
			if fv.Len() > 0 {
				fv = fv.Index(0)
				return mapstructure.DecodeHookExec(fc, fv, tv)
			}
			return nil, nil
		}
		return data, nil
	}
}

// BinaryUnmarshalerHookFunc returns a DecodeHookFunc that applies
// strings to the UnmarshalBinary function, when the target type
// implements the encoding.BinaryUnmarshaler interface
func BinaryUnmarshalerHookFunc() mapstructure.DecodeHookFuncType {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		result := reflect.New(t).Interface()
		unmarshaller, ok := result.(encoding.BinaryUnmarshaler)
		if !ok {
			return data, nil
		}
		if err := unmarshaller.UnmarshalBinary([]byte(data.(string))); err != nil {
			return nil, err
		}
		return result, nil
	}
}
