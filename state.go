// Read and write the state file used to keep data over multiple runs

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
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/google/renameio"
)

type State struct {
	// Key is File.Url
	LastModified map[string]time.Time
	Checksum     map[string]string // SHA512 in hex
}

func LoadState(path string) (*State, error) {
	var state State

	x, err := ioutil.ReadFile(path)
	if err != nil {
		// It's fine if the state file does not exist yet, we'll
		// create it later when writing the state
		if !os.IsNotExist(err) {
			return nil, err
		}
	} else {
		err := json.Unmarshal(x, &state)
		if err != nil {
			return nil, err
		}
	}

	if state.LastModified == nil {
		state.LastModified = make(map[string]time.Time)
	}
	if state.Checksum == nil {
		state.Checksum = make(map[string]string)
	}

	return &state, nil
}

func WriteState(path string, state *State) error {
	// Update the state file even if nothing has changed to provide a
	// simple way to check if nsscash ran successfully (the state is only
	// updated if there were no errors)

	x, err := json.Marshal(state)
	if err != nil {
		return err
	}

	f, err := renameio.TempFile(filepath.Dir(path), path)
	if err != nil {
		return err
	}
	defer f.Cleanup()

	_, err = f.Write(x)
	if err != nil {
		return err
	}
	err = f.CloseAtomicallyReplace()
	if err != nil {
		return err
	}
	return syncPath(filepath.Dir(path))
}
