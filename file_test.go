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
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestDeployFile(t *testing.T) {
	const deploy = "testdata/deploy"

	// Suppress log messages
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	// Setup & cleanup
	f, err := os.Create(deploy)
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(deploy)

	tests := []struct {
		perm os.FileMode
		exp  os.FileMode
	}{
		{
			0644,
			0444,
		},
		{
			0400,
			0400,
		},
		{
			0600,
			0400,
		},
		{
			0777,
			0555,
		},
		{
			0666,
			0444,
		},
		{
			0000,
			0000,
		},
	}

	for n, tc := range tests {
		err = os.Chmod(deploy, tc.perm)
		if err != nil {
			t.Fatal(err)
		}
		err = deployFile(&File{
			Type: FileTypePlain,
			Url:  "http://example.org",
			Path: deploy,
			body: []byte("Hello World!"),
		})
		if err != nil {
			t.Fatal(err)
		}

		info, err := os.Stat(deploy)
		if err != nil {
			t.Fatal(err)
		}
		perm := info.Mode().Perm()
		if perm != tc.exp {
			t.Errorf("%d: perm = %v, want %v",
				n, perm, tc.exp)
		}
	}
}
