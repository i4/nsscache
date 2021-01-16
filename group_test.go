// Copyright (C) 2019-2021  Simon Ruderich
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestParseGroups(t *testing.T) {
	tests := []struct {
		data string
		exp  []Group
		err  error
	}{
		{
			"",
			nil,
			nil,
		},
		{
			"root:x:0:",
			nil,
			fmt.Errorf("no newline in last line: \"root:x:0:\""),
		},
		{
			"root:x:0:\n",
			[]Group{
				Group{
					Name:    "root",
					Passwd:  "x",
					Gid:     0,
					Members: nil,
				},
			},
			nil,
		},
		{
			"root:x:0:foo\n",
			[]Group{
				Group{
					Name:   "root",
					Passwd: "x",
					Gid:    0,
					Members: []string{
						"foo",
					},
				},
			},
			nil,
		},
		{
			"root:x:0:foo,bar\n",
			[]Group{
				Group{
					Name:   "root",
					Passwd: "x",
					Gid:    0,
					Members: []string{
						"foo",
						"bar",
					},
				},
			},
			nil,
		},
	}

	for n, tc := range tests {
		res, err := ParseGroups(strings.NewReader(tc.data))
		if !reflect.DeepEqual(err, tc.err) {
			t.Errorf("%d: err = %v, want %v",
				n, err, tc.err)
		}
		if !reflect.DeepEqual(res, tc.exp) {
			t.Errorf("%d: res = %v, want %v",
				n, res, tc.exp)
		}
	}
}
