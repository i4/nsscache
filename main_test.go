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
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

const (
	configPath  = "testdata/config.toml"
	statePath   = "testdata/var/state.json"
	passwdPath  = "testdata/passwd.nsscash"
	plainPath   = "testdata/plain"
	groupPath   = "testdata/group.nsscash"
	tlsCAPath   = "testdata/ca.crt"
	tlsCertPath = "testdata/server.crt"
	tlsKeyPath  = "testdata/server.key"
	tlsCA2Path  = "testdata/ca2.crt"
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

func hashAsHex(x []byte) string {
	h := sha1.New()
	h.Write(x)
	return hex.EncodeToString(h.Sum(nil))
}

// mustHaveHash checks if the given path content has the given SHA-1 string
// (in hex).
func mustHaveHash(t *testing.T, path string, hash string) {
	x, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	y := hashAsHex(x)
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
ca = "%[4]s"
`, statePath, url, passwdPath, tlsCAPath))
}

func mustWriteGroupConfig(t *testing.T, url string) {
	mustWriteConfig(t, fmt.Sprintf(`
statepath = "%[1]s"

[[file]]
type = "group"
url = "%[2]s/group"
path = "%[3]s"
ca = "%[4]s"
`, statePath, url, groupPath, tlsCAPath))
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

// mustBeOld verifies that all paths have a modification time in the past, as
// set by mustMakeOld.
func mustBeOld(t *testing.T, paths ...string) {
	for _, p := range paths {
		i, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}

		mtime := i.ModTime()
		if time.Since(mtime) < time.Hour {
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
		if time.Since(mtime) > time.Hour {
			t.Errorf("%q was not recently modified", p)
		}
	}
}

func TestMainFetch(t *testing.T) {
	// Suppress log messages
	log.SetOutput(ioutil.Discard)
	defer log.SetOutput(os.Stderr)

	tests := []func(args){
		// Perform most tests with passwd for simplicity
		fetchPasswdCacheFileDoesNotExist,
		fetchPasswd404,
		fetchPasswdUnexpected304,
		fetchPasswdEmpty,
		fetchPasswdInvalid,
		fetchPasswdLimits,
		fetchPasswd,
		// Tests for plain and group
		fetchPlainEmpty,
		fetchPlain,
		fetchGroupEmpty,
		fetchGroupInvalid,
		fetchGroupLimits,
		fetchGroup,
		// Special tests
		fetchNoConfig,
		fetchStateCannotRead,
		fetchStateInvalid,
		fetchStateCannotWrite,
		fetchCannotDeploy,
		fetchSecondFetchFails,
		fetchBasicAuth,
	}

	// HTTP tests

	for _, f := range tests {
		runMainTest(t, f, nil)
	}

	// HTTPS tests

	tests = append(tests, fetchInvalidCA)

	cert, err := tls.LoadX509KeyPair(tlsCertPath, tlsKeyPath)
	if err != nil {
		t.Fatal(err)
	}
	tls := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	for _, f := range tests {
		runMainTest(t, f, tls)
	}
}

func runMainTest(t *testing.T, f func(args), tls *tls.Config) {
	cleanup := []string{
		configPath,
		statePath,
		passwdPath,
		plainPath,
		groupPath,
	}

	// NOTE: This is not guaranteed to work according to reflect's
	// documentation but seems to work reliable for normal functions.
	fn := runtime.FuncForPC(reflect.ValueOf(f).Pointer())
	name := fn.Name()
	name = name[strings.LastIndex(name, ".")+1:]
	if tls != nil {
		name = "tls" + name
	}

	t.Run(name, func(t *testing.T) {
		// Preparation & cleanup
		for _, p := range cleanup {
			err := os.Remove(p)
			if err != nil && !os.IsNotExist(err) {
				t.Fatal(err)
			}
			// Remove the file at the end of this test run, if it
			// was created
			defer os.Remove(p)

			dir := filepath.Dir(p)
			err = os.MkdirAll(dir, 0755)
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(dir) // remove empty directories
		}

		var handler func(http.ResponseWriter, *http.Request)
		ts := httptest.NewUnstartedServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				handler(w, r)
			}))
		if tls == nil {
			ts.Start()
		} else {
			ts.TLS = tls
			ts.StartTLS()
		}
		defer ts.Close()

		f(args{
			t:       t,
			url:     ts.URL,
			handler: &handler,
		})
	})
}

func fetchPasswdCacheFileDoesNotExist(a args) {
	t := a.t
	mustWritePasswdConfig(t, a.url)

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"file.path \""+passwdPath+"\" must exist")

	mustNotExist(t, statePath, passwdPath, plainPath, groupPath)
}

func fetchPasswd404(a args) {
	t := a.t
	mustWritePasswdConfig(t, a.url)
	mustCreate(t, passwdPath)

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		// 404
		w.WriteHeader(http.StatusNotFound)
	}

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"status code 404")

	mustNotExist(t, statePath, plainPath, groupPath)
	mustBeOld(t, passwdPath)
}

func fetchPasswdUnexpected304(a args) {
	t := a.t
	mustWritePasswdConfig(t, a.url)
	mustCreate(t, passwdPath)

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		// 304
		w.WriteHeader(http.StatusNotModified)
	}

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"status code 304 but did not send If-Modified-Since")

	mustNotExist(t, statePath, plainPath, groupPath)
	mustBeOld(t, passwdPath)
}

func fetchPasswdEmpty(a args) {
	t := a.t
	mustWritePasswdConfig(t, a.url)
	mustCreate(t, passwdPath)

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		// Empty response
	}

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"refusing to use empty passwd file")

	mustNotExist(t, statePath, plainPath, groupPath)
	mustBeOld(t, passwdPath)
}

func fetchPasswdInvalid(a args) {
	t := a.t
	mustWritePasswdConfig(t, a.url)
	mustCreate(t, passwdPath)

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/passwd" {
			return
		}

		fmt.Fprintln(w, "root:x:invalid:0:root:/root:/bin/bash")
	}

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"invalid uid in line")

	mustNotExist(t, statePath, plainPath, groupPath)
	mustBeOld(t, passwdPath)
}

func fetchPasswdLimits(a args) {
	t := a.t
	mustWritePasswdConfig(t, a.url)
	mustCreate(t, passwdPath)

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/passwd" {
			return
		}

		fmt.Fprint(w, "root:x:0:0:root:/root:/bin/bash")
		for i := 0; i < 65536; i++ {
			fmt.Fprint(w, "x")
		}
		fmt.Fprint(w, "\n")
	}

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"passwd too large to serialize")

	mustNotExist(t, statePath, plainPath, groupPath)
	mustBeOld(t, passwdPath)
}

func fetchPasswd(a args) {
	t := a.t
	mustWritePasswdConfig(t, a.url)
	mustCreate(t, passwdPath)
	mustHaveHash(t, passwdPath, "da39a3ee5e6b4b0d3255bfef95601890afd80709")

	t.Log("First fetch, write files")

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/passwd" {
			return
		}

		// No "Last-Modified" header
		fmt.Fprintln(w, "root:x:0:0:root:/root:/bin/bash")
		fmt.Fprintln(w, "daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin")
	}

	err := mainFetch(configPath)
	if err != nil {
		t.Error(err)
	}

	mustNotExist(t, plainPath, groupPath)
	mustBeNew(t, passwdPath, statePath)
	// The actual content of passwdPath is verified by the NSS tests
	mustHaveHash(t, passwdPath, "bbb7db67469b111200400e2470346d5515d64c23")

	t.Log("Fetch again, no support for Last-Modified")

	mustMakeOld(t, passwdPath, statePath)

	err = mainFetch(configPath)
	if err != nil {
		t.Error(err)
	}

	mustNotExist(t, plainPath, groupPath)
	mustBeNew(t, passwdPath, statePath)
	mustHaveHash(t, passwdPath, "bbb7db67469b111200400e2470346d5515d64c23")

	t.Log("Fetch again, support for Last-Modified, but not retrieved yet")

	mustMakeOld(t, passwdPath, statePath)

	lastChange := time.Now()
	change := false
	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/passwd" {
			return
		}

		modified := r.Header.Get("If-Modified-Since")
		if modified != "" {
			x, err := http.ParseTime(modified)
			if err != nil {
				t.Fatalf("invalid If-Modified-Since %v",
					modified)
			}
			if !x.Before(lastChange.Truncate(time.Second)) {
				w.WriteHeader(http.StatusNotModified)
				return
			}
		}

		w.Header().Add("Last-Modified",
			lastChange.UTC().Format(http.TimeFormat))
		fmt.Fprintln(w, "root:x:0:0:root:/root:/bin/bash")
		fmt.Fprintln(w, "daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin")
		if change {
			fmt.Fprintln(w, "bin:x:2:2:bin:/bin:/usr/sbin/nologin")
		}
	}

	err = mainFetch(configPath)
	if err != nil {
		t.Error(err)
	}

	mustNotExist(t, plainPath, groupPath)
	mustBeNew(t, passwdPath, statePath)
	mustHaveHash(t, passwdPath, "bbb7db67469b111200400e2470346d5515d64c23")

	t.Log("Fetch again, support for Last-Modified")

	mustMakeOld(t, passwdPath, statePath)

	err = mainFetch(configPath)
	if err != nil {
		t.Error(err)
	}

	mustNotExist(t, plainPath, groupPath)
	mustBeOld(t, passwdPath)
	mustBeNew(t, statePath)
	mustHaveHash(t, passwdPath, "bbb7db67469b111200400e2470346d5515d64c23")

	t.Log("Corrupt local passwd cache, fetched again")

	os.Chmod(passwdPath, 0644) // make writable again
	mustCreate(t, passwdPath)
	mustMakeOld(t, passwdPath, statePath)

	err = mainFetch(configPath)
	if err != nil {
		t.Error(err)
	}

	mustNotExist(t, plainPath, groupPath)
	mustBeNew(t, passwdPath, statePath)
	mustHaveHash(t, passwdPath, "bbb7db67469b111200400e2470346d5515d64c23")

	t.Log("Fetch again with newer server response")

	change = true
	lastChange = time.Now().Add(time.Second)

	mustMakeOld(t, passwdPath, statePath)

	err = mainFetch(configPath)
	if err != nil {
		t.Error(err)
	}

	mustNotExist(t, plainPath, groupPath)
	mustBeNew(t, passwdPath, statePath)
	mustHaveHash(t, passwdPath, "ca9c7477cb425667fc9ecbd79e8e1c2ad0e84423")
}

func fetchPlainEmpty(a args) {
	t := a.t
	mustWriteConfig(t, fmt.Sprintf(`
statepath = "%[1]s"

[[file]]
type = "plain"
url = "%[2]s/plain"
path = "%[3]s"
ca = "%[4]s"
`, statePath, a.url, plainPath, tlsCAPath))
	mustCreate(t, plainPath)

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		// Empty response
	}

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"refusing to use empty response")

	mustNotExist(t, statePath, passwdPath, groupPath)
	mustBeOld(t, plainPath)
}

func fetchPlain(a args) {
	t := a.t
	mustWriteConfig(t, fmt.Sprintf(`
statepath = "%[1]s"

[[file]]
type = "plain"
url = "%[2]s/plain"
path = "%[3]s"
ca = "%[4]s"
`, statePath, a.url, plainPath, tlsCAPath))
	mustCreate(t, plainPath)
	mustHaveHash(t, plainPath, "da39a3ee5e6b4b0d3255bfef95601890afd80709")

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/plain" {
			return
		}

		fmt.Fprintln(w, "some file")
	}

	err := mainFetch(configPath)
	if err != nil {
		t.Error(err)
	}

	mustNotExist(t, passwdPath, groupPath)
	mustBeNew(t, plainPath, statePath)
	mustHaveHash(t, plainPath, "0e08b5e8c10abc3e455b75286ba4a1fbd56e18a5")

	// Remaining functionality already tested in fetchPasswd()
}

func fetchGroupEmpty(a args) {
	t := a.t
	mustWriteGroupConfig(t, a.url)
	mustCreate(t, groupPath)

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		// Empty response
	}

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"refusing to use empty group file")

	mustNotExist(t, statePath, passwdPath, plainPath)
	mustBeOld(t, groupPath)
}

func fetchGroupInvalid(a args) {
	t := a.t
	mustWriteGroupConfig(t, a.url)
	mustCreate(t, groupPath)

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/group" {
			return
		}

		fmt.Fprintln(w, "root:x::")
	}

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"invalid gid in line")

	mustNotExist(t, statePath, passwdPath, plainPath)
	mustBeOld(t, groupPath)
}

func fetchGroupLimits(a args) {
	t := a.t
	mustWriteGroupConfig(t, a.url)
	mustCreate(t, groupPath)

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/group" {
			return
		}

		fmt.Fprint(w, "root:x:0:")
		for i := 0; i < 65536; i++ {
			fmt.Fprint(w, "x")
		}
		fmt.Fprint(w, "\n")
	}

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"group too large to serialize")

	mustNotExist(t, statePath, passwdPath, plainPath)
	mustBeOld(t, groupPath)
}

func fetchGroup(a args) {
	t := a.t
	mustWriteGroupConfig(t, a.url)
	mustCreate(t, groupPath)
	mustHaveHash(t, groupPath, "da39a3ee5e6b4b0d3255bfef95601890afd80709")

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/group" {
			return
		}

		fmt.Fprintln(w, "root:x:0:")
		fmt.Fprintln(w, "daemon:x:1:andariel,duriel,mephisto,diablo,baal")
	}

	err := mainFetch(configPath)
	if err != nil {
		t.Error(err)
	}

	mustNotExist(t, passwdPath, plainPath)
	mustBeNew(t, groupPath, statePath)
	// The actual content of groupPath is verified by the NSS tests
	mustHaveHash(t, groupPath, "8c27a8403278ba2e392b86d98d4dff1fdefcafdd")

	// Remaining functionality already tested in fetchPasswd()
}

func fetchNoConfig(a args) {
	t := a.t

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		configPath+": no such file or directory")

	mustNotExist(t, configPath, statePath, passwdPath, plainPath, groupPath)
}

func fetchStateCannotRead(a args) {
	t := a.t
	mustWritePasswdConfig(t, a.url)

	mustCreate(t, statePath)
	err := os.Chmod(statePath, 0000)
	if err != nil {
		t.Fatal(err)
	}

	err = mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		statePath+": permission denied")

	mustNotExist(t, passwdPath, plainPath, groupPath)
	mustBeOld(t, statePath)
}

func fetchStateInvalid(a args) {
	t := a.t
	mustWriteGroupConfig(t, a.url)
	mustCreate(t, statePath)

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"unexpected end of JSON input")

	mustNotExist(t, groupPath, passwdPath, plainPath)
	mustBeOld(t, statePath)
}

func fetchStateCannotWrite(a args) {
	t := a.t
	mustWriteGroupConfig(t, a.url)
	mustCreate(t, groupPath)
	mustHaveHash(t, groupPath, "da39a3ee5e6b4b0d3255bfef95601890afd80709")

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/group" {
			return
		}

		fmt.Fprintln(w, "root:x:0:")
		fmt.Fprintln(w, "daemon:x:1:andariel,duriel,mephisto,diablo,baal")
	}

	err := os.Chmod(filepath.Dir(statePath), 0500)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(filepath.Dir(statePath), 0755)

	err = mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"permission denied")

	mustNotExist(t, statePath, passwdPath, plainPath)
	mustBeNew(t, groupPath)
	mustHaveHash(t, groupPath, "8c27a8403278ba2e392b86d98d4dff1fdefcafdd")
}

func fetchCannotDeploy(a args) {
	t := a.t
	mustWriteGroupConfig(t, a.url)
	mustCreate(t, groupPath)
	mustHaveHash(t, groupPath, "da39a3ee5e6b4b0d3255bfef95601890afd80709")

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/group" {
			return
		}

		fmt.Fprintln(w, "root:x:0:")
		fmt.Fprintln(w, "daemon:x:1:andariel,duriel,mephisto,diablo,baal")
	}

	err := os.Chmod("testdata", 0500)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chmod("testdata", 0755)

	err = mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"permission denied")

	mustNotExist(t, statePath, passwdPath, plainPath)
	mustBeOld(t, groupPath)
}

func fetchSecondFetchFails(a args) {
	t := a.t
	mustWriteConfig(t, fmt.Sprintf(`
statepath = "%[1]s"

[[file]]
type = "passwd"
url = "%[2]s/passwd"
path = "%[3]s"
ca = "%[5]s"

[[file]]
type = "group"
url = "%[2]s/group"
path = "%[4]s"
ca = "%[5]s"
`, statePath, a.url, passwdPath, groupPath, tlsCAPath))
	mustCreate(t, passwdPath)
	mustCreate(t, groupPath)
	mustHaveHash(t, passwdPath, "da39a3ee5e6b4b0d3255bfef95601890afd80709")
	mustHaveHash(t, groupPath, "da39a3ee5e6b4b0d3255bfef95601890afd80709")

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/passwd" {
			fmt.Fprintln(w, "root:x:0:0:root:/root:/bin/bash")
		}
		if r.URL.Path == "/group" {
			w.WriteHeader(http.StatusNotFound)
		}
	}

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"status code 404")

	mustNotExist(t, statePath, plainPath)
	// Even though passwd was successfully fetched, no files were modified
	// because the second fetch failed
	mustBeOld(t, passwdPath, groupPath)
}

func fetchBasicAuth(a args) {
	t := a.t
	mustWritePasswdConfig(t, a.url)
	mustCreate(t, passwdPath)
	mustHaveHash(t, passwdPath, "da39a3ee5e6b4b0d3255bfef95601890afd80709")

	validUser := "username"
	validPass := "password"

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/passwd" {
			return
		}

		user, pass, ok := r.BasicAuth()
		// NOTE: Do not use this in production because it permits
		// attackers to determine the length of user/pass. Instead use
		// hashes and subtle.ConstantTimeCompare().
		if !ok || user != validUser || pass != validPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="Test"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		fmt.Fprintln(w, "root:x:0:0:root:/root:/bin/bash")
		fmt.Fprintln(w, "daemon:x:1:1:daemon:/usr/sbin:/usr/sbin/nologin")
	}

	t.Log("Missing authentication")

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"status code 401")

	mustNotExist(t, statePath, groupPath, plainPath)
	mustBeOld(t, passwdPath)

	t.Log("Unsafe config permissions")

	mustWriteConfig(t, fmt.Sprintf(`
statepath = "%[1]s"

[[file]]
type = "passwd"
url = "%[2]s/passwd"
path = "%[3]s"
ca = "%[4]s"
username = "%[5]s"
password = "%[6]s"
`, statePath, a.url, passwdPath, tlsCAPath, validUser, validPass))

	err = os.Chmod(configPath, 0644)
	if err != nil {
		t.Fatal(err)
	}

	err = mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"file[0].username/passsword in use and unsafe permissions "+
			"-rw-r--r-- on \"testdata/config.toml\"")

	mustNotExist(t, statePath, groupPath, plainPath)
	mustBeOld(t, passwdPath)

	t.Log("Working authentication")

	err = os.Chmod(configPath, 0600)
	if err != nil {
		t.Fatal(err)
	}

	err = mainFetch(configPath)
	if err != nil {
		t.Error(err)
	}

	mustNotExist(t, plainPath, groupPath)
	mustBeNew(t, passwdPath, statePath)
	mustHaveHash(t, passwdPath, "bbb7db67469b111200400e2470346d5515d64c23")
}

func fetchInvalidCA(a args) {
	t := a.t

	// System CA

	mustWriteConfig(t, fmt.Sprintf(`
statepath = "%[1]s"

[[file]]
type = "passwd"
url = "%[2]s/passwd"
path = "%[3]s"
`, statePath, a.url, passwdPath))
	mustCreate(t, passwdPath)
	mustHaveHash(t, passwdPath, "da39a3ee5e6b4b0d3255bfef95601890afd80709")

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/passwd" {
			fmt.Fprintln(w, "root:x:0:0:root:/root:/bin/bash")
		}
	}

	err := mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"x509: certificate signed by unknown authority")

	mustNotExist(t, statePath, plainPath, groupPath)
	mustBeOld(t, passwdPath)

	// Invalid CA

	mustWriteConfig(t, fmt.Sprintf(`
statepath = "%[1]s"

[[file]]
type = "passwd"
url = "%[2]s/passwd"
path = "%[3]s"
ca = "%[4]s"
`, statePath, a.url, passwdPath, tlsCA2Path))
	mustCreate(t, passwdPath)
	mustHaveHash(t, passwdPath, "da39a3ee5e6b4b0d3255bfef95601890afd80709")

	*a.handler = func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/passwd" {
			fmt.Fprintln(w, "root:x:0:0:root:/root:/bin/bash")
		}
	}

	err = mainFetch(configPath)
	mustBeErrorWithSubstring(t, err,
		"x509: certificate signed by unknown authority")

	mustNotExist(t, statePath, plainPath, groupPath)
	mustBeOld(t, passwdPath)
}
