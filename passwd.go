// Parse /etc/passwd files and serialize them

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
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Version written in SerializePasswds()
const PasswdVersion = 1

type Passwd struct {
	Name   string
	Passwd string
	Uid    uint64
	Gid    uint64
	Gecos  string
	Dir    string
	Shell  string
}

// ParsePasswds parses a file in the format of /etc/passwd and returns all
// entries as slice of Passwd structs.
func ParsePasswds(r io.Reader) ([]Passwd, error) {
	var res []Passwd

	s := bufio.NewReader(r)
	for {
		t, err := s.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		x := strings.Split(t, ":")
		if len(x) != 7 {
			return nil, fmt.Errorf("invalid line %q", t)
		}

		uid, err := strconv.ParseUint(x[2], 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid uid in line %q", t)
		}
		gid, err := strconv.ParseUint(x[3], 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid gid in line %q", t)
		}

		res = append(res, Passwd{
			Name:   x[0],
			Passwd: x[1],
			Uid:    uid,
			Gid:    gid,
			Gecos:  x[4],
			Dir:    x[5],
			// ReadString() contains the delimiter
			Shell: strings.TrimSuffix(x[6], "\n"),
		})
	}
	return res, nil
}

func SerializePasswd(p Passwd) ([]byte, error) {
	// Concatenate all (NUL-terminated) strings and store the offsets
	var data bytes.Buffer
	data.Write([]byte(p.Name))
	data.WriteByte(0)
	offPasswd := uint16(data.Len())
	data.Write([]byte(p.Passwd))
	data.WriteByte(0)
	offGecos := uint16(data.Len())
	data.Write([]byte(p.Gecos))
	data.WriteByte(0)
	offDir := uint16(data.Len())
	data.Write([]byte(p.Dir))
	data.WriteByte(0)
	offShell := uint16(data.Len())
	data.Write([]byte(p.Shell))
	data.WriteByte(0)
	// Ensure the offsets can fit the length of this entry
	if data.Len() > math.MaxUint16 {
		return nil, fmt.Errorf("passwd too large to serialize: %v, %v",
			data.Len(), p)
	}
	size := uint16(data.Len())

	var res bytes.Buffer // serialized result
	le := binary.LittleEndian

	id := make([]byte, 8)
	// uid
	le.PutUint64(id, p.Uid)
	res.Write(id)
	// gid
	le.PutUint64(id, p.Gid)
	res.Write(id)

	off := make([]byte, 2)
	// off_passwd
	le.PutUint16(off, offPasswd)
	res.Write(off)
	// off_gecos
	le.PutUint16(off, offGecos)
	res.Write(off)
	// off_dir
	le.PutUint16(off, offDir)
	res.Write(off)
	// off_shell
	le.PutUint16(off, offShell)
	res.Write(off)
	// data_size
	le.PutUint16(off, size)
	res.Write(off)

	res.Write(data.Bytes())
	// We must pad each entry so that all uint64 at the beginning of the
	// struct are 8 byte aligned
	alignBufferTo(&res, 8)

	return res.Bytes(), nil
}

func SerializePasswds(w io.Writer, pws []Passwd) error {
	// Serialize passwords and store offsets
	var data bytes.Buffer
	offsets := make(map[Passwd]uint64)
	for _, p := range pws {
		// TODO: warn about duplicate entries
		offsets[p] = uint64(data.Len())
		x, err := SerializePasswd(p)
		if err != nil {
			return err
		}
		data.Write(x)
	}

	// Copy to prevent sorting from modifying the argument
	sorted := make([]Passwd, len(pws))
	copy(sorted, pws)

	le := binary.LittleEndian
	tmp := make([]byte, 8)

	// Create index "sorted" in input order, used when iterating over all
	// passwd entries (getpwent_r); keeping the original order makes
	// debugging easier
	var indexOrig bytes.Buffer
	for _, p := range pws {
		le.PutUint64(tmp, offsets[p])
		indexOrig.Write(tmp)
	}

	// Create index sorted after id
	var indexId bytes.Buffer
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Uid < sorted[j].Uid
	})
	for _, p := range sorted {
		le.PutUint64(tmp, offsets[p])
		indexId.Write(tmp)
	}

	// Create index sorted after name
	var indexName bytes.Buffer
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	for _, p := range sorted {
		le.PutUint64(tmp, offsets[p])
		indexName.Write(tmp)
	}

	// Sanity check
	if len(pws)*8 != indexOrig.Len() ||
		indexOrig.Len() != indexId.Len() ||
		indexId.Len() != indexName.Len() {
		return fmt.Errorf("indexes have inconsistent length")
	}

	// Write result

	// magic
	w.Write([]byte("NSS-CASH"))
	// version
	le.PutUint64(tmp, PasswdVersion)
	w.Write(tmp)
	// count
	le.PutUint64(tmp, uint64(len(pws)))
	w.Write(tmp)
	// off_orig_index
	offset := uint64(0)
	le.PutUint64(tmp, offset)
	w.Write(tmp)
	// off_id_index
	offset += uint64(indexOrig.Len())
	le.PutUint64(tmp, offset)
	w.Write(tmp)
	// off_name_index
	offset += uint64(indexId.Len())
	le.PutUint64(tmp, offset)
	w.Write(tmp)
	// off_data
	offset += uint64(indexName.Len())
	le.PutUint64(tmp, offset)
	w.Write(tmp)

	_, err := indexOrig.WriteTo(w)
	if err != nil {
		return err
	}
	_, err = indexId.WriteTo(w)
	if err != nil {
		return err
	}
	_, err = indexName.WriteTo(w)
	if err != nil {
		return err
	}
	_, err = data.WriteTo(w)
	if err != nil {
		return err
	}

	return nil
}
