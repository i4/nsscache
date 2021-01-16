// Download and write files atomically to the file system

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
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/google/renameio"
	"github.com/pkg/errors"
)

func handleFiles(cfg *Config, state *State) error {
	// Split into fetch and deploy phase to prevent updates of only some
	// files which might lead to inconsistent state; obviously this won't
	// work during the deploy phase, but it helps if the web server fails
	// to deliver some files

	for i, f := range cfg.Files {
		err := fetchFile(&cfg.Files[i], state)
		if err != nil {
			return errors.Wrapf(err, "%q (%v)", f.Url, f.Type)
		}
	}

	for i, f := range cfg.Files {
		// No update required
		if f.body == nil {
			continue
		}

		err := deployFile(&cfg.Files[i])
		if err != nil {
			return errors.Wrapf(err, "%q (%v)", f.Url, f.Type)
		}
	}

	return nil
}

func checksumFile(file *File) (string, error) {
	x, err := ioutil.ReadFile(file.Path)
	if err != nil {
		return "", err
	}
	return checksumBytes(x), nil
}

func checksumBytes(x []byte) string {
	h := sha512.New()
	h.Write(x)
	return hex.EncodeToString(h.Sum(nil))
}

func fetchFile(file *File, state *State) error {
	t := state.LastModified[file.Url]

	hash, err := checksumFile(file)
	if err != nil {
		// See below in deployFile() for the reason
		return errors.Wrapf(err, "file.path %q must exist", file.Path)
	}
	if hash != state.Checksum[file.Url] {
		log.Printf("%q -> %q: hash has changed", file.Url, file.Path)
		var zero time.Time
		t = zero // force download
	}

	oldT := t
	status, body, err := fetchIfModified(file.Url,
		file.Username, file.Password, file.CA, &t)
	if err != nil {
		return err
	}
	if status == http.StatusNotModified {
		if oldT.IsZero() {
			return fmt.Errorf("status code 304 " +
				"but did not send If-Modified-Since")
		}
		log.Printf("%q -> %q: not modified", file.Url, file.Path)
		return nil
	}
	if status != http.StatusOK {
		return fmt.Errorf("status code %v", status)
	}
	state.LastModified[file.Url] = t

	if file.Type == FileTypePlain {
		if len(body) == 0 {
			return fmt.Errorf("refusing to use empty response")
		}
		file.body = body

	} else if file.Type == FileTypePasswd {
		pws, err := ParsePasswds(bytes.NewReader(body))
		if err != nil {
			return err
		}
		// Safety check: having no users can be very dangerous, don't
		// permit it
		if len(pws) == 0 {
			return fmt.Errorf("refusing to use empty passwd file")
		}

		var x bytes.Buffer
		err = SerializePasswds(&x, pws)
		if err != nil {
			return err
		}
		file.body = x.Bytes()

	} else if file.Type == FileTypeGroup {
		grs, err := ParseGroups(bytes.NewReader(body))
		if err != nil {
			return err
		}
		if len(grs) == 0 {
			return fmt.Errorf("refusing to use empty group file")
		}

		var x bytes.Buffer
		err = SerializeGroups(&x, grs)
		if err != nil {
			return err
		}
		file.body = x.Bytes()

	} else {
		return fmt.Errorf("unsupported file type %v", file.Type)
	}

	state.Checksum[file.Url] = checksumBytes(file.body)
	return nil
}

func deployFile(file *File) error {
	log.Printf("%q -> %q: updating file", file.Url, file.Path)

	// Safety check
	if len(file.body) == 0 {
		return fmt.Errorf("refusing to write empty file")
	}

	f, err := renameio.TempFile(filepath.Dir(file.Path), file.Path)
	if err != nil {
		return err
	}
	defer f.Cleanup()

	// Apply permissions/user/group from the target file but remove the
	// write permissions to discourage manual modifications, use Stat
	// instead of Lstat as only the target's permissions are relevant
	stat, err := os.Stat(file.Path)
	if err != nil {
		// We do not create the path if it doesn't exist, because we
		// do not know the proper permissions
		return errors.Wrapf(err, "file.path %q must exist", file.Path)
	}
	err = f.Chmod(stat.Mode() & ^os.FileMode(0222)) // remove write perms
	if err != nil {
		return err
	}
	// TODO: support more systems
	sys, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("unsupported FileInfo.Sys()")
	}
	err = f.Chown(int(sys.Uid), int(sys.Gid))
	if err != nil {
		return err
	}

	_, err = f.Write(file.body)
	if err != nil {
		return err
	}
	err = f.CloseAtomicallyReplace()
	if err != nil {
		return err
	}
	return syncPath(filepath.Dir(file.Path))
}
