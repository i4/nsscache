// Download and write files atomically to the file system

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
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
)

func handleFiles(cfg *Config, state *State) error {
	for i, f := range cfg.Files {
		err := fetchFile(&cfg.Files[i], state)
		if err != nil {
			return errors.Wrapf(err, "%q (%s)", f.Url, f.Type)
		}
	}

	for i, f := range cfg.Files {
		// No update required
		if f.body == nil {
			continue
		}

		err := deployFile(&cfg.Files[i])
		if err != nil {
			return errors.Wrapf(err, "%q (%s)", f.Url, f.Type)
		}
	}

	return nil
}

func fetchFile(file *File, state *State) error {
	t := state.LastModified[file.Url]
	status, body, err := fetchIfModified(file.Url, &t)
	if err != nil {
		return err
	}
	if status == http.StatusNotModified {
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

	} else {
		return fmt.Errorf("unsupported file type %v", file.Type)
	}
	return nil
}

func deployFile(file *File) error {
	log.Printf("%q -> %q: updating file", file.Url, file.Path)

	// Safety check
	if len(file.body) == 0 {
		return fmt.Errorf("refusing to write empty file")
	}

	// Write the file in an atomic fashion by creating a temporary file
	// and renaming it over the target file

	dir := filepath.Dir(file.Path)
	name := filepath.Base(file.Path)

	f, err := ioutil.TempFile(dir, "tmp-"+name+"-")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	defer f.Close()

	// Apply permissions/user/group from the target file
	stat, err := os.Stat(file.Path)
	if err != nil {
		// We do not create the path if it doesn't exist, because we
		// do not know the proper permissions
		return errors.Wrapf(err, "file.path %q must exist", file.Path)
	}
	err = f.Chmod(stat.Mode())
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
	err = f.Sync()
	if err != nil {
		return err
	}
	return os.Rename(f.Name(), file.Path)
}
