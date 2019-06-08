// Main file for nsscash

// Copyright (C) 2019  Simon Ruderich
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
	"log"
	"os"
)

func main() {
	if len(os.Args) != 2 {
		log.SetFlags(0)
		log.Fatalf("usage: %s <path/to/config>\n", os.Args[0])
	}

	cfg, err := LoadConfig(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	state, err := LoadState(cfg.StatePath)
	if err != nil {
		log.Fatal(err)
	}

	err = handleFiles(cfg, state)
	if err != nil {
		log.Fatal(err)
	}

	err = WriteStateIfChanged(cfg.StatePath, state)
	if err != nil {
		log.Fatal(err)
	}
}
