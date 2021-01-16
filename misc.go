// Miscellaneous functions

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
	"bytes"
	"fmt"
	"os"
)

func alignBufferTo(b *bytes.Buffer, align int) {
	if align <= 0 {
		panic(fmt.Sprintf("invalid alignment %v", align))
	}

	l := b.Len()
	if l%align == 0 {
		return
	}
	for i := 0; i < align-l%align; i++ {
		b.WriteByte(0)
	}
}

// syncPath syncs path, which should be a directory. To guarantee durability
// it must be called on a parent directory after adding, renaming or removing
// files therein.
//
// Calling sync on the files itself is not enough according to POSIX; man 2
// fsync: "Calling fsync() does not necessarily ensure that the entry in the
// directory containing the file has also reached disk. For that an explicit
// fsync() on a file descriptor for the directory is also needed."
func syncPath(path string) error {
	x, err := os.Open(path)
	if err != nil {
		return err
	}
	err = x.Sync()
	closeErr := x.Close()
	if err != nil {
		return err
	}
	return closeErr
}
