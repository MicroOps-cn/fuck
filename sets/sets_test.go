//go:build go1.18

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

package sets

import (
	"reflect"
	"testing"
)

func TestSet(t *testing.T) {
	t.Run("Test string set", func(t *testing.T) {
		stringSet := New[string]("aa", "A1", "x11", "11", ".", "asdj03y4hr3eotdfh", "A1", "-asohdd")
		t.Run("Test List", func(t *testing.T) {
			wantList := []string{"aa", "A1", "x11", "11", ".", "asdj03y4hr3eotdfh", "-asohdd"}
			if !reflect.DeepEqual(stringSet.List(), wantList) {
				t.Errorf("List() = %v, want %v", stringSet.List(), wantList)
			}
		})
		t.Run("Test SortedList", func(t *testing.T) {
			wantList := []string{"-asohdd", ".", "11", "A1", "aa", "asdj03y4hr3eotdfh", "x11"}
			if !reflect.DeepEqual(stringSet.SortedList(), wantList) {
				t.Errorf("SortedList() = %v, want %v", stringSet.SortedList(), wantList)
			}
		})
		t.Run("Test Has", func(t *testing.T) {
			if !reflect.DeepEqual(stringSet.Has("asdj03y4hr3eotdfh"), true) {
				t.Errorf("Has() = %v, want %v", stringSet.Has("asdj03y4hr3eotdfh"), true)
			}
			if !reflect.DeepEqual(stringSet.Has("AA"), false) {
				t.Errorf("Has() = %v, want %v", stringSet.Has("AA"), false)
			}
		})
		t.Run("Test Equal", func(t *testing.T) {
			stringSet2 := New[string]("-asohdd", "asdj03y4hr3eotdfh", "x11", ".", "11", "A1", "aa")
			if !reflect.DeepEqual(stringSet.Equal(stringSet2), true) {
				t.Errorf("Equal() = %v, want %v", stringSet.Equal(stringSet2), true)
			}
			stringSet3 := New[string]("-asohdd", "asdj03y4hr3eotdfh", "x112", ".", "11", "A1", "aa")
			if !reflect.DeepEqual(stringSet.Equal(stringSet3), false) {
				t.Errorf("Equal() = %v, want %v", stringSet.Equal(stringSet3), false)
			}
		})
		t.Run("Test Len", func(t *testing.T) {
			if !reflect.DeepEqual(stringSet.Len(), 7) {
				t.Errorf("Len() = %v, want %v", stringSet.Len(), 7)
			}
		})
		t.Run("Test Delete", func(t *testing.T) {
			newStringList := stringSet.Clone().Delete("11", "A1", ".").List()
			wantList := []string{"aa", "x11", "asdj03y4hr3eotdfh", "-asohdd"}
			if !reflect.DeepEqual(newStringList, wantList) {
				t.Errorf("Delete() = %v, want %v", newStringList, wantList)
			}
		})
		t.Run("Test Difference", func(t *testing.T) {
			newStringList := stringSet.Difference(New[string]("aa", "x11", "m2", "asdj03y4hr3eotdfh", "-asohdd")).SortedList()
			wantList := []string{".", "11", "A1"}
			if !reflect.DeepEqual(newStringList, wantList) {
				t.Errorf("Difference() = %v, want %v", newStringList, wantList)
			}
		})
		t.Run("Test Union", func(t *testing.T) {
			newStringList := stringSet.Union(New[string]("m2", "asdj03y4hr3eotdfh", "qq", "-asohdd")).SortedList()
			wantList := []string{"-asohdd", ".", "11", "A1", "aa", "asdj03y4hr3eotdfh", "m2", "qq", "x11"}
			if !reflect.DeepEqual(newStringList, wantList) {
				t.Errorf("Union() = %v, want %v", newStringList, wantList)
			}
		})
		t.Run("Test Insert", func(t *testing.T) {
			newStringList := stringSet.Clone().Insert("AA", "11", "aa", "bb").SortedList()
			wantList := []string{"-asohdd", ".", "11", "A1", "AA", "aa", "asdj03y4hr3eotdfh", "bb", "x11"}
			if !reflect.DeepEqual(newStringList, wantList) {
				t.Errorf("Insert() = %v, want %v", newStringList, wantList)
			}
		})
		t.Run("Test HasAny", func(t *testing.T) {
			hasAny := stringSet.HasAny("AA", "11", "aa", "bb")
			if !reflect.DeepEqual(hasAny, true) {
				t.Errorf("HasAny() = %v, want %v", hasAny, true)
			}
			hasAny = stringSet.HasAny("bb", "q123")
			if !reflect.DeepEqual(hasAny, false) {
				t.Errorf("HasAny() = %v, want %v", hasAny, false)
			}
		})

		t.Run("Test HasAll", func(t *testing.T) {
			hasAll := stringSet.HasAll("AA", "11", "aa", "bb")
			if !reflect.DeepEqual(hasAll, false) {
				t.Errorf("HasAll() = %v, want %v", hasAll, false)
			}
			hasAll = stringSet.HasAll("A1", "11", "aa")
			if !reflect.DeepEqual(hasAll, true) {
				t.Errorf("HasAll() = %v, want %v", hasAll, true)
			}
		})

		t.Run("Test IsSuperset", func(t *testing.T) {
			isSuperset := stringSet.IsSuperset(New[string]("aa", "A1", ".", "x11"))
			if !reflect.DeepEqual(isSuperset, true) {
				t.Errorf("Has() = %v, want %v", isSuperset, true)
			}
			isSuperset = stringSet.IsSuperset(New[string]("aa", "A1", "ax", "x11"))
			if !reflect.DeepEqual(isSuperset, false) {
				t.Errorf("Has() = %v, want %v", isSuperset, false)
			}
		})

		t.Run("Test Intersection", func(t *testing.T) {
			newSet := stringSet.Intersection(New[string]("ax", ".", "AA", "aa", "A1")).SortedList()
			wantList := []string{".", "A1", "aa"}
			if !reflect.DeepEqual(newSet, wantList) {
				t.Errorf("Has() = %v, want %v", newSet, wantList)
			}
		})
	})

	t.Run("Test int64 set", func(t *testing.T) {
		stringSet := New[int64](1, 83, -123, 84389, 99, 99, 83, -121038)
		t.Run("Test List", func(t *testing.T) {
			wantList := []int64{1, 83, -123, 84389, 99, -121038}
			if !reflect.DeepEqual(stringSet.List(), wantList) {
				t.Errorf("List() = %v, want %v", stringSet.List(), wantList)
			}
		})
		t.Run("Test SortedList", func(t *testing.T) {
			wantList := []int64{-121038, -123, 1, 83, 99, 84389}
			if !reflect.DeepEqual(stringSet.SortedList(), wantList) {
				t.Errorf("SortedList() = %v, want %v", stringSet.SortedList(), wantList)
			}
		})
		t.Run("Test Has", func(t *testing.T) {
			if !reflect.DeepEqual(stringSet.Has(99), true) {
				t.Errorf("Has() = %v, want %v", stringSet.Has(99), true)
			}
			if !reflect.DeepEqual(stringSet.Has(100), false) {
				t.Errorf("Has() = %v, want %v", stringSet.Has(100), false)
			}
		})
		t.Run("Test Equal", func(t *testing.T) {
			stringSet2 := New[int64](1, 83, -123, 84389, 99, -123, -121038)
			if !reflect.DeepEqual(stringSet.Equal(stringSet2), true) {
				t.Errorf("Equal() = %v, want %v", stringSet.Equal(stringSet2), true)
			}
			stringSet3 := New[int64](2, 83, -123, 84389, 99, -123, -121038)
			if !reflect.DeepEqual(stringSet.Equal(stringSet3), false) {
				t.Errorf("Equal() = %v, want %v", stringSet.Equal(stringSet3), false)
			}
		})
		t.Run("Test Len", func(t *testing.T) {
			if !reflect.DeepEqual(stringSet.Len(), 6) {
				t.Errorf("Len() = %v, want %v", stringSet.Len(), 6)
			}
		})
		t.Run("Test Delete", func(t *testing.T) {
			newStringList := stringSet.Clone().Delete(83, 99, -123).List()
			wantList := []int64{1, 84389, -121038}
			if !reflect.DeepEqual(newStringList, wantList) {
				t.Errorf("Delete() = %v, want %v", newStringList, wantList)
			}
		})
		t.Run("Test Difference", func(t *testing.T) {
			newStringList := stringSet.Difference(New[int64](1, 83, -123, 66, 99, 91, -121038)).SortedList()
			wantList := []int64{84389}
			if !reflect.DeepEqual(newStringList, wantList) {
				t.Errorf("Difference() = %v, want %v", newStringList, wantList)
			}
		})
		t.Run("Test Union", func(t *testing.T) {
			newStringList := stringSet.Union(New[int64](1, 2, 3)).SortedList()
			wantList := []int64{-121038, -123, 1, 2, 3, 83, 99, 84389}
			if !reflect.DeepEqual(newStringList, wantList) {
				t.Errorf("Union() = %v, want %v", newStringList, wantList)
			}
		})
		t.Run("Test Insert", func(t *testing.T) {
			newStringList := stringSet.Clone().Insert(7, -11, 2).SortedList()
			wantList := []int64{-121038, -123, -11, 1, 2, 7, 83, 99, 84389}
			if !reflect.DeepEqual(newStringList, wantList) {
				t.Errorf("Insert() = %v, want %v", newStringList, wantList)
			}
		})
		t.Run("Test HasAny", func(t *testing.T) {
			hasAny := stringSet.HasAny(1, 3, 4)
			if !reflect.DeepEqual(hasAny, true) {
				t.Errorf("HasAny() = %v, want %v", hasAny, true)
			}
			hasAny = stringSet.HasAny(0, 2, 3)
			if !reflect.DeepEqual(hasAny, false) {
				t.Errorf("HasAny() = %v, want %v", hasAny, false)
			}
		})

		t.Run("Test HasAll", func(t *testing.T) {
			hasAll := stringSet.HasAll(1, 99)
			if !reflect.DeepEqual(hasAll, true) {
				t.Errorf("HasAll() = %v, want %v", hasAll, true)
			}
			hasAll = stringSet.HasAll(1, 2, 99)
			if !reflect.DeepEqual(hasAll, false) {
				t.Errorf("HasAll() = %v, want %v", hasAll, false)
			}
		})

		t.Run("Test IsSuperset", func(t *testing.T) {
			isSuperset := stringSet.IsSuperset(New[int64](1, 99, -123))
			if !reflect.DeepEqual(isSuperset, true) {
				t.Errorf("Has() = %v, want %v", isSuperset, true)
			}
			isSuperset = stringSet.IsSuperset(New[int64](1, 2, 99, -123))
			if !reflect.DeepEqual(isSuperset, false) {
				t.Errorf("Has() = %v, want %v", isSuperset, false)
			}
		})

		t.Run("Test Intersection", func(t *testing.T) {
			newSet := stringSet.Intersection(New[int64](1, 2, 3, 99)).SortedList()
			wantList := []int64{1, 99}
			if !reflect.DeepEqual(newSet, wantList) {
				t.Errorf("Has() = %v, want %v", newSet, wantList)
			}
		})
	})
}
