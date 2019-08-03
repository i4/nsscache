// Configuration file parsing and validation

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
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	StatePath string
	Files     []File `toml:"file"`
}

type File struct {
	Type     FileType
	Url      string
	Path     string
	CA       string
	Username string
	Password string

	body []byte // internally used by handleFiles()
}

//go:generate stringer -type=FileType
type FileType int

const (
	FileTypePlain FileType = iota
	FileTypePasswd
	FileTypeGroup
)

func (t *FileType) UnmarshalText(text []byte) error {
	switch string(text) {
	case "plain":
		*t = FileTypePlain
	case "passwd":
		*t = FileTypePasswd
	case "group":
		*t = FileTypeGroup
	default:
		return fmt.Errorf("invalid file type %q", text)
	}
	return nil
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config

	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, err
	}
	undecoded := md.Undecoded()
	if len(undecoded) != 0 {
		return nil, fmt.Errorf("invalid fields used: %q", undecoded)
	}

	f, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	perms := f.Mode().Perm()
	unsafe := (perms & 0077) != 0 // readable by others

	if cfg.StatePath == "" {
		return nil, fmt.Errorf("statepath must not be empty")
	}

	for i, f := range cfg.Files {
		if f.Url == "" {
			return nil, fmt.Errorf(
				"file[%d].url must not be empty", i)
		}
		if f.Path == "" {
			return nil, fmt.Errorf(
				"file[%d].path must not be empty", i)
		}
		if (f.Username != "" || f.Password != "") && unsafe {
			return nil, fmt.Errorf(
				"file[%d].username/passsword in use and "+
					"unsafe permissions %v on %q",
				i, perms, path)
		}
	}

	return &cfg, nil
}
