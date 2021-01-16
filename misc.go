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
