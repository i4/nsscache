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
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	configPath = "testdata/config.toml"
	statePath  = "testdata/state.json"
	passwdPath = "testdata/passwd.nsscash"
	plainPath  = "testdata/plain"
	groupPath  = "testdata/group.nsscash"
)

type args struct {
	t       *testing.T
	url     string
	handler *func(http.ResponseWriter, *http.Request)
}

// mustNotExist verifies that all given paths don't exist in the file system.
func mustNotExist(t *testing.T, paths ...string) {
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			if !os.IsNotExist(err) {
				t.Errorf("path %q: unexpected error: %v",
					p, err)
			}
		} else {
			t.Errorf("path %q exists", p)
			f.Close()
		}
	}
}

// mustHaveHash checks if the given path content has the given SHA-1 string
// (in hex).
func mustHaveHash(t *testing.T, path string, hash string) {
	x, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	h := sha1.New()
	h.Write(x)
	y := hex.EncodeToString(h.Sum(nil))

	if y != hash {
		t.Errorf("%q has unexpected hash %q", path, y)
	}
}

// mustBeErrorWithSubstring checks if the given error, represented as string,
// contains the given substring. This is somewhat ugly but the simplest way to
// check for proper errors.
func mustBeErrorWithSubstring(t *testing.T, err error, substring string) {
	if err == nil {
		t.Errorf("err is nil")
	} else if !strings.Contains(err.Error(), substring) {
		t.Errorf("err %q does not contain string %q", err, substring)
	}
}

func mustWriteConfig(t *testing.T, config string) {
	err := ioutil.WriteFile(configPath, []byte(config), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func mustWritePasswdConfig(t *testing.T, url string) {
	mustWriteConfig(t, fmt.Sprintf(`
statepath = "%[1]s"

[[file]]
type = "passwd"
url = "%[2]s/passwd"
path = "%[3]s"
`, statePath, url, passwdPath))
}

func mustWriteGroupConfig(t *testing.T, url string) {
	mustWriteConfig(t, fmt.Sprintf(`
statepath = "%[1]s"

[[file]]
type = "group"
url = "%[2]s/group"
path = "%[3]s"
`, statePath, url, groupPath))
}

// mustCreate creates a file, truncating it if it exists. It then changes the
// modification to be in the past.
func mustCreate(t *testing.T, path string) {
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	err = f.Close()
	if err != nil {
		t.Fatal(err)
	}

	// Change modification time to the past to detect updates to the file
	mustMakeOld(t, path)
}

// mustMakeOld change the modification time of all paths to be in the past.
func mustMakeOld(t *testing.T, paths ...string) {
	old := time.Now().Add(-2 * time.Hour)
	for _, p := range paths {
		err := os.Chtimes(p, old, old)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// mustMakeOld verifies that all paths have a modification time in the past,
// as set by mustMakeOld().
func mustBeOld(t *testing.T, paths ...string) {
	for _, p := range paths {
		i, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}

		mtime := i.ModTime()
		now := time.Now()
		if now.Sub(mtime) < time.Hour {
			t.Errorf("%q was recently modified", p)
		}
	}
}

// mustBeNew verifies that all paths have a modification time in the present.
func mustBeNew(t *testing.T, paths ...string) {
	for _, p := range paths {
		i, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}

		mtime := i.ModTime()
		now := time.Now()
		if now.Sub(mtime) > time.Hour {
			t.Errorf("%q was not recently modified", p)
		}
	}
}

func TestMainFetch(t *testing.T) {
	// Suppress log messages
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	tests := []func(args){
		// Special tests
		fetchNoConfig,
	}

	cleanup := []string{
		configPath,
		statePath,
		passwdPath,
		plainPath,
		groupPath,
	}

	for _, f := range tests {
		// NOTE: This is not guaranteed to work according to reflect's
		// documentation but seems to work reliable for normal
		// functions.
		fn := runtime.FuncForPC(reflect.ValueOf(f).Pointer())
		name := fn.Name()
		name = name[strings.LastIndex(name, ".")+1:]

		t.Run(name, func(t *testing.T) {
			// Preparation & cleanup
			for _, p := range cleanup {
				err := os.Remove(p)
				if err != nil && !os.IsNotExist(err) {
					t.Fatal(err)
				}
				// Remove the file at the end of this test
				// run, if it was created
				defer os.Remove(p)
			}

			var handler func(http.ResponseWriter, *http.Request)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handler(w, r)
			}))
			defer ts.Close()

			f(args{
				t:       t,
				url:     ts.URL,
				handler: &handler,
			})
		})
	}
}

func fetchNoConfig(a args) {
	t := a.t

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		configPath+": no such file or directory")

	mustNotExist(t, configPath, statePath, passwdPath, plainPath, groupPath)
}
