// Read and write the state file used to keep data over multiple runs

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
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

type State struct {
	LastModified map[string]time.Time
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

	// Write the file in an atomic fashion by creating a temporary file
	// and renaming it over the target file

	dir := filepath.Dir(path)
	name := filepath.Base(path)

	f, err := ioutil.TempFile(dir, "tmp-"+name+"-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()

	_, err = f.Write(x)
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
