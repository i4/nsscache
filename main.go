// Main file for nsscash

// Copyright (C) 2019-2020  Simon Ruderich
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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/google/renameio"
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr,
			"usage: %[1]s [options] fetch <config>\n"+
				"usage: %[1]s [options] convert <type> <src> <dst>\n"+
				"",
			os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	switch args[0] {
	case "fetch":
		if len(args) != 2 {
			break
		}

		err := mainFetch(args[1])
		if err != nil {
			log.Fatal(err)
		}
		return

	case "convert":
		if len(args) != 4 {
			break
		}

		err := mainConvert(args[1], args[2], args[3])
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	flag.Usage()
	os.Exit(1)
}

func mainFetch(cfgPath string) error {
	cfg, err := LoadConfig(cfgPath)
	if err != nil {
		return err
	}
	state, err := LoadState(cfg.StatePath)
	if err != nil {
		return err
	}
	err = handleFiles(cfg, state)
	if err != nil {
		return err
	}
	// NOTE: Make sure to call WriteState() only if there were no
	// errors (see WriteState() and README)
	err = WriteState(cfg.StatePath, state)
	if err != nil {
		return err
	}
	return nil
}

func mainConvert(typ, srcPath, dstPath string) error {
	var t FileType
	err := t.UnmarshalText([]byte(typ))
	if err != nil {
		return err
	}

	src, err := ioutil.ReadFile(srcPath)
	if err != nil {
		return err
	}
	var x bytes.Buffer
	if t == FileTypePlain {
		x.Write(src)
	} else if t == FileTypePasswd {
		pws, err := ParsePasswds(bytes.NewReader(src))
		if err != nil {
			return err
		}
		err = SerializePasswds(&x, pws)
		if err != nil {
			return err
		}
	} else if t == FileTypeGroup {
		grs, err := ParseGroups(bytes.NewReader(src))
		if err != nil {
			return err
		}
		err = SerializeGroups(&x, grs)
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("unsupported file type %v", t)
	}

	// We must create the file first or deployFile() will abort; this is
	// ugly because deployFile() already performs an atomic replacement
	// but the simplest solution with the least duplicate code
	f, err := renameio.TempFile(filepath.Dir(dstPath), dstPath)
	if err != nil {
		return err
	}
	defer f.Cleanup()

	err = deployFile(&File{
		Type: t,
		Url:  srcPath,
		Path: f.Name(),
		body: x.Bytes(),
	})
	if err != nil {
		return err
	}

	return f.CloseAtomicallyReplace()
}
