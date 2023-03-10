/*
 * Copyright © 2022 MicroOps-cn.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package capacity

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var unitMap = map[string]int64{
	"b":  int64(Byte),
	"kb": int64(Kilobyte),
	"mb": int64(Megabyte),
	"gb": int64(Gigabyte),
	"tb": int64(Terabyte),
	"k":  int64(Kilobyte),
	"m":  int64(Megabyte),
	"g":  int64(Gigabyte),
	"t":  int64(Terabyte),
}

func quote(s string) string {
	return "\"" + s + "\""
}

// leadingFraction consumes the leading [0-9]* from s.
// It is used only for fractions, so does not return an error on overflow,
// it just stops accumulating precision.
func leadingFraction(s string) (x int64, scale float64, rem string) {
	i := 0
	scale = 1
	overflow := false
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if overflow {
			continue
		}
		if x > (1<<63-1)/10 {
			// It's possible for overflow to give a positive number, so take care.
			overflow = true
			continue
		}
		y := x*10 + int64(c) - '0'
		if y < 0 {
			overflow = true
			continue
		}
		x = y
		scale *= 10
	}
	return x, scale, s[i:]
}

var errLeadingInt = errors.New("capacity: bad [0-9]*") // never printed
func leadingInt(s string) (x int64, rem string, err error) {
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if x > (1<<63-1)/10 {
			// overflow
			return 0, "", errLeadingInt
		}
		x = x*10 + int64(c) - '0'
		if x < 0 {
			// overflow
			return 0, "", errLeadingInt
		}
	}
	return x, s[i:], nil
}

func ParseCapacities(s string) (Capacities, error) {
	var d int64
	orig := s
	neg := false
	if s != "" {
		c := s[0]
		if c == '-' || c == '+' {
			neg = c == '-'
			s = s[1:]
		}
	}
	if s == "0" {
		return 0, nil
	}
	if s == "" {
		return 0, errors.New("capacity: invalid capacity " + quote(orig))
	}
	for s != "" {
		var (
			v, f  int64       // integers before, after decimal point
			scale float64 = 1 // value = v + f/scale
		)

		var err error

		// The next character must be [0-9.]
		if !(s[0] == '.' || '0' <= s[0] && s[0] <= '9') {
			return 0, errors.New("capacity: invalid capacity " + quote(orig))
		}
		// Consume [0-9]*
		pl := len(s)
		v, s, err = leadingInt(s)
		if err != nil {
			return 0, fmt.Errorf("capacity: invalid capacity %s: %s", quote(orig), err)
		}
		pre := pl != len(s) // whether we consumed anything before a period

		// Consume (\.[0-9]*)?
		post := false
		if s != "" && s[0] == '.' {
			s = s[1:]
			pl := len(s)
			f, scale, s = leadingFraction(s)
			post = pl != len(s)
		}
		if !pre && !post {
			// no digits (e.g. ".s" or "-.s")
			return 0, errors.New("capacity: invalid capacity " + quote(orig))
		}

		// Consume unit.
		i := 0
		for ; i < len(s); i++ {
			c := s[i]
			if c == '.' || '0' <= c && c <= '9' {
				break
			}
		}
		if i == 0 {
			return 0, errors.New("capacity: missing unit in capacity " + quote(orig))
		}
		u := s[:i]
		s = s[i:]
		unit, ok := unitMap[strings.ToLower(u)]
		if !ok {
			return 0, errors.New("capacity: unknown unit " + quote(u) + " in capacity " + quote(orig))
		}
		if v > (1<<63-1)/unit {
			// overflow
			return 0, fmt.Errorf("capacity: invalid capacity %s: %s", quote(orig), "overflow")
		}
		v *= unit
		if f > 0 {
			// float64 is needed to be nanosecond accurate for fractions of hours.
			// v >= 0 && (f*unit/scale) <= 3.6e+12 (ns/h, h is the largest unit)
			v += int64(float64(f) * (float64(unit) / scale))
			if v < 0 {
				// overflow
				return 0, errors.New("capacity: invalid capacity " + quote(orig))
			}
		}
		d += v
		if d < 0 {
			// overflow
			return 0, errors.New("capacity: invalid capacity " + quote(orig))
		}
	}

	if neg {
		d = -d
	}
	return Capacities(d), nil
}

func (c *Capacities) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		return nil
	} else if len(b) > 2 && b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	} else {
		val, err := strconv.ParseInt(string(b), 10, 64)
		*c = Capacities(val)
		return err
	}

	var err error
	*c, err = ParseCapacities(string(b))
	return err
}

func (c Capacities) MarshalJSON() ([]byte, error) {
	b := append([]byte{'"'}, c.String()...)
	b = append(b, '"')
	return b, nil
}

func (c *Capacities) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	var err error
	*c, err = ParseCapacities(s)
	return err
}

func (c Capacities) MarshalYAML() (interface{}, error) {
	return c.String(), nil
}

// StringToCapacityHookFunc Implementation mapstructure.DecodeHookFunc
func StringToCapacityHookFunc() func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{},
	) (interface{}, error) {
		if f.Kind() != reflect.String {
			return data, nil
		}
		if t == reflect.TypeOf(Capacities(5)) {
			return ParseCapacities(data.(string))
		}
		return data, nil
	}
}
