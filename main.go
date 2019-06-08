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
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
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

		cfg, err := LoadConfig(args[1])
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
		return

	case "convert":
		if len(args) != 4 {
			break
		}

		var t FileType
		err := t.UnmarshalText([]byte(args[1]))
		if err != nil {
			log.Fatal(err)
		}

		src, err := ioutil.ReadFile(args[2])
		if err != nil {
			log.Fatal(err)
		}
		var x bytes.Buffer
		if t == FileTypePlain {
			x.Write(src)
		} else if t == FileTypePasswd {
		pws, err := ParsePasswds(bytes.NewReader(src))
		if err != nil {
			log.Fatal(err)
		}
		err = SerializePasswds(&x, pws)
		if err != nil {
			log.Fatal(err)
		}
		} else {
			log.Fatalf("unsupported file type %v", t)
		}

		// We must create the file first or deployFile() will abort
		f, err := os.Create(args[3])
		if err != nil {
			log.Fatal(err)
		}
		f.Close()

		err = deployFile(&File{
			Type: t,
			Url:  args[2],
			Path: args[3],
			body: x.Bytes(),
		})
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	flag.Usage()
	os.Exit(1)
}
