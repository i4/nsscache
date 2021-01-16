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

func TestParsePasswd(t *testing.T) {
	tests := []struct {
		data string
		exp  []Passwd
		err  error
	}{
		{
			"",
			nil,
			nil,
		},
		{
			"root:x:0:0:root:/root:/bin/zsh",
			nil,
			fmt.Errorf("no newline in last line: \"root:x:0:0:root:/root:/bin/zsh\""),
		},
		{
			"root:x:0:0:root:/root:/bin/zsh\n",
			[]Passwd{
				Passwd{
					Name:   "root",
					Passwd: "x",
					Uid:    0,
					Gid:    0,
					Gecos:  "root",
					Dir:    "/root",
					Shell:  "/bin/zsh",
				},
			},
			nil,
		},
		{
			"root:x:0:0:root:/root:/bin/zsh\n" +
				"daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin\n" +
				"bin:x:2:3:bin:/bin:/usr/sbin/nologin\n",
			[]Passwd{
				Passwd{
					Name:   "root",
					Passwd: "x",
					Uid:    0,
					Gid:    0,
					Gecos:  "root",
					Dir:    "/root",
					Shell:  "/bin/zsh",
				},
				Passwd{
					Name:   "daemon",
					Passwd: "x",
					Uid:    1,
					Gid:    1,
					Gecos:  "daemon",
					Dir:    "/usr/sbin",
					Shell:  "/usr/sbin/nologin",
				},
				Passwd{
					Name:   "bin",
					Passwd: "x",
					Uid:    2,
					Gid:    3,
					Gecos:  "bin",
					Dir:    "/bin",
					Shell:  "/usr/sbin/nologin",
				},
			},
			nil,
		},
	}

	for n, tc := range tests {
		res, err := ParsePasswds(strings.NewReader(tc.data))
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
