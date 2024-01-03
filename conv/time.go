package conv

import (
	"encoding/json"
	w "github.com/MicroOps-cn/fuck/wrapper"
	"strconv"
	"time"
)

const (
	TimeFormatUnixTimestamp                 = "unix-timestamp"
	TimeFormatUnixMilliTimestamp            = "unix-milli-timestamp"
	TimeFormatUnixTimestampString           = "unix-timestamp-string"
	TimeFormatUnixMilliTimestampString      = "unix-milli-timestamp-string"
	TimeFormatFloatUnixTimestamp            = "float-unix-timestamp"
	TimeFormatFloatUnixMilliTimestamp       = "float-unix-milli-timestamp"
	TimeFormatFloatUnixTimestampString      = "float-unix-timestamp-string"
	TimeFormatFloatUnixMilliTimestampString = "float-unix-milli-timestamp-string"
)

type Time struct {
	time.Time
	Format string
}

func (t Time) MarshalJSON() ([]byte, error) {
	switch t.Format {
	case "":
		return json.Marshal(t.Time)
	case TimeFormatUnixTimestamp:
		return []byte(strconv.FormatInt(t.Unix(), 10)), nil
	case TimeFormatUnixTimestampString:
		return w.M(json.Marshal(strconv.FormatInt(t.Unix(), 10))), nil
	case TimeFormatUnixMilliTimestamp:
		return []byte(strconv.FormatInt(t.UnixMilli(), 10)), nil
	case TimeFormatFloatUnixMilliTimestampString:
		return w.M(json.Marshal(strconv.FormatFloat(float64(t.UnixMilli()), 'f', -1, 64))), nil
	case TimeFormatFloatUnixTimestamp:
		return []byte(strconv.FormatFloat(float64(t.UnixMilli())/1e3, 'f', -1, 64)), nil
	case TimeFormatFloatUnixTimestampString:
		return w.M(json.Marshal(strconv.FormatFloat(float64(t.UnixMilli())/1e3, 'f', -1, 64))), nil
	case TimeFormatFloatUnixMilliTimestamp:
		return []byte(strconv.FormatFloat(float64(t.UnixMilli()), 'f', -1, 64)), nil
	case TimeFormatUnixMilliTimestampString:
		return w.M(json.Marshal(strconv.FormatFloat(float64(t.UnixMilli()), 'f', -1, 64))), nil
	default:
		return json.Marshal(t.Time.Format(t.Format))
	}
}

var timeFormats = []string{
	time.DateTime, time.RFC3339, time.RFC3339Nano, time.DateOnly, time.Layout, time.ANSIC,
	time.UnixDate, time.RubyDate, time.RFC822, time.RFC822Z, time.RFC850, time.RFC1123, time.RFC1123Z,
}

func (t *Time) fromTimestampStrng(raw string) bool {
	if ts, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if ts > 1e12 {
			t.Time = time.UnixMilli(ts)
			t.Format = TimeFormatUnixMilliTimestamp
			return true
		}
		t.Time = time.Unix(ts, 0)
		t.Format = TimeFormatUnixTimestamp
		return true
	}
	if ts, err := strconv.ParseFloat(raw, 64); err == nil {
		if ts > 1e12 {
			t.Format = TimeFormatFloatUnixMilliTimestamp
			t.Time = time.UnixMilli(int64(ts))
			return true
		}
		t.Time = time.Unix(int64(ts), int64((ts-float64(int64(ts)))*1e3)%1e3*1e6)
		t.Format = TimeFormatFloatUnixTimestamp
		return true
	}
	return false
}
func (t *Time) UnmarshalJSON(raw []byte) error {
	if t.fromTimestampStrng(string(raw)) {
		return nil
	}
	var timeStr string
	if err := json.Unmarshal(raw, &timeStr); err != nil {
		return err
	}
	if t.fromTimestampStrng(timeStr) {
		t.Format += "-string"
		return nil
	}
	var err error
	for _, timeFormat := range timeFormats {
		t.Time, err = time.Parse(timeFormat, timeStr)
		if err == nil {
			t.Format = timeFormat
			break
		}
	}
	return err
}
