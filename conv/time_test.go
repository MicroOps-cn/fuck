package conv

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestTime_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name           string
		args           string
		wantErr        bool
		want           Time
		disableMarshal bool
	}{{
		name: "DateTime",
		args: `"2023-01-02 11:22:33"`,
		want: Time{
			Time:   time.Date(2023, 01, 02, 11, 22, 33, 0, time.UTC),
			Format: time.DateTime,
		},
	}, {
		name: "DateOnly",
		args: `"2023-01-02"`,
		want: Time{
			Time:   time.Date(2023, 01, 02, 0, 0, 0, 0, time.UTC),
			Format: time.DateOnly,
		},
	}, {
		name: "RFC3339",
		args: `"2023-01-02T11:22:33+09:00"`,
		want: Time{
			Time:   time.Date(2023, 01, 02, 11, 22, 33, 0, time.FixedZone("", 9*60*60)),
			Format: time.RFC3339,
		},
	}, {
		name: "unix timestamp",
		args: `1704189127`,
		want: Time{
			Time:   time.Date(2024, 01, 2, 17, 52, 7, 0, time.Local),
			Format: TimeFormatUnixTimestamp,
		},
	}, {
		name: "unix Mill timestamp",
		args: `1704189127123`,
		want: Time{
			Time:   time.Date(2024, 01, 2, 17, 52, 7, 123*1e6, time.Local),
			Format: TimeFormatUnixMilliTimestamp,
		},
	}, {
		name: "unix timestamp string",
		args: `"1704189127"`,
		want: Time{
			Time:   time.Date(2024, 01, 2, 17, 52, 7, 0, time.Local),
			Format: TimeFormatUnixTimestampString,
		},
	}, {
		name: "unix Mill timestamp string",
		args: `"1704189127123"`,
		want: Time{
			Time:   time.Date(2024, 01, 2, 17, 52, 7, 123*1e6, time.Local),
			Format: TimeFormatUnixMilliTimestampString,
		},
	}, {
		name: "float unix timestamp",
		args: `1704189127.5`,
		want: Time{
			Time:   time.Date(2024, 01, 2, 17, 52, 7, 500*1e6, time.Local),
			Format: TimeFormatFloatUnixTimestamp,
		},
	}, {
		name: "float unix Mill timestamp",
		args: `1704189127123.0`,
		want: Time{
			Time:   time.Date(2024, 01, 2, 17, 52, 7, 123*1e6, time.Local),
			Format: TimeFormatFloatUnixMilliTimestamp,
		},
		disableMarshal: true,
	}, {
		name: "float unix timestamp string",
		args: `"1704189127.5"`,
		want: Time{
			Time:   time.Date(2024, 01, 2, 17, 52, 7, 500*1e6, time.Local),
			Format: TimeFormatFloatUnixTimestampString,
		},
	}, {
		name: "float unix Mill timestamp string",
		args: `"1704189127123.0"`,
		want: Time{
			Time:   time.Date(2024, 01, 2, 17, 52, 7, 123*1e6, time.Local),
			Format: TimeFormatFloatUnixMilliTimestampString,
		},
		disableMarshal: true,
	}, {
		name: "error string",
		args: `{}`,
		want: Time{
			Time:   time.Date(2024, 01, 2, 17, 52, 7, 123*1e6, time.Local),
			Format: TimeFormatUnixMilliTimestampString,
		},
		wantErr: true,
	}, {
		name: "error format",
		args: `"2023-01-02T11:2:33+09:00"`,
		want: Time{
			Time:   time.Date(2024, 01, 2, 17, 52, 7, 123*1e6, time.Local),
			Format: TimeFormatUnixMilliTimestampString,
		},
		wantErr: true,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t1 *testing.T) {
			tm := &Time{}
			if err := json.Unmarshal([]byte(tt.args), tm); (err != nil) != tt.wantErr {
				t1.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			} else if !tt.wantErr {
				require.Equal(t1, tt.want, *tm)
				if !tt.disableMarshal {

					raw, err := json.Marshal(tm)
					require.NoError(t1, err)
					require.Equal(t1, string(raw), tt.args)
				}
			}
		})
	}
}
