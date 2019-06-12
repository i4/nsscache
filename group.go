// Parse /etc/group files and serialize them

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

// Version written in SerializeGroups()
const GroupVersion = 1

type Group struct {
	Name    string
	Passwd  string
	Gid     uint64
	Members []string
}

// Go does not support slices in map keys; therefore, use this separate struct
type GroupKey struct {
	Name    string
	Passwd  string
	Gid     uint64
	Members string // "," separated
}

func toKey(g Group) GroupKey {
	return GroupKey{
		Name:    g.Name,
		Passwd:  g.Passwd,
		Gid:     g.Gid,
		Members: strings.Join(g.Members, ","),
	}
}

// ParseGroups parses a file in the format of /etc/group and returns all
// entries as slice of Group structs.
func ParseGroups(r io.Reader) ([]Group, error) {
	var res []Group

	s := bufio.NewReader(r)
	for {
		t, err := s.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				if t != "" {
					return nil, fmt.Errorf(
						"no newline in last line: %q",
						t)
				}
				break
			}
			return nil, err
		}

		x := strings.Split(t, ":")
		if len(x) != 4 {
			return nil, fmt.Errorf("invalid line %q", t)
		}

		gid, err := strconv.ParseUint(x[2], 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid gid in line %q", t)
		}

		// ReadString() contains the delimiter
		x[3] = strings.TrimSuffix(x[3], "\n")

		var members []string
		// No members must result in empty slice, not slice with the
		// empty string
		if x[3] != "" {
			members = strings.Split(x[3], ",")
		}
		res = append(res, Group{
			Name:    x[0],
			Passwd:  x[1],
			Gid:     gid,
			Members: members,
		})
	}

	return res, nil
}

func SerializeGroup(g Group) ([]byte, error) {
	le := binary.LittleEndian

	// Concatenate all (NUL-terminated) strings and store the offsets
	var mems bytes.Buffer
	var mems_off []uint16
	for _, m := range g.Members {
		mems_off = append(mems_off, uint16(mems.Len()))
		mems.Write([]byte(m))
		mems.WriteByte(0)
	}
	var data bytes.Buffer
	data.Write([]byte(g.Name))
	data.WriteByte(0)
	offPasswd := uint16(data.Len())
	data.Write([]byte(g.Passwd))
	data.WriteByte(0)
	alignBufferTo(&data, 2) // align the following uint16
	offMemOff := uint16(data.Len())
	// Offsets for group members
	offMem := offMemOff + 2*uint16(len(mems_off))
	for _, o := range mems_off {
		tmp := make([]byte, 2)
		le.PutUint16(tmp, offMem+o)
		data.Write(tmp)
	}
	// And the group members concatenated as above
	data.Write(mems.Bytes())
	// Ensure the offsets can fit the length of this entry
	if data.Len() > math.MaxUint16 {
		return nil, fmt.Errorf("group too large to serialize: %v, %v",
			data.Len(), g)
	}
	size := uint16(data.Len())

	var res bytes.Buffer // serialized result

	id := make([]byte, 8)
	// gid
	le.PutUint64(id, g.Gid)
	res.Write(id)

	off := make([]byte, 2)
	// off_passwd
	le.PutUint16(off, offPasswd)
	res.Write(off)
	// off_mem_off
	le.PutUint16(off, offMemOff)
	res.Write(off)
	// mem_count
	le.PutUint16(off, uint16(len(g.Members)))
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

func SerializeGroups(w io.Writer, grs []Group) error {
	// Serialize groups and store offsets
	var data bytes.Buffer
	offsets := make(map[GroupKey]uint64)
	for _, g := range grs {
		// TODO: warn about duplicate entries
		offsets[toKey(g)] = uint64(data.Len())
		x, err := SerializeGroup(g)
		if err != nil {
			return err
		}
		data.Write(x)
	}

	// Copy to prevent sorting from modifying the argument
	sorted := make([]Group, len(grs))
	copy(sorted, grs)

	le := binary.LittleEndian
	tmp := make([]byte, 8)

	// Create index "sorted" in input order, used when iterating over all
	// passwd entries (getgrent_r); keeping the original order makes
	// debugging easier
	var indexOrig bytes.Buffer
	for _, g := range grs {
		le.PutUint64(tmp, offsets[toKey(g)])
		indexOrig.Write(tmp)
	}

	// Create index sorted after id
	var indexId bytes.Buffer
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Gid < sorted[j].Gid
	})
	for _, g := range sorted {
		le.PutUint64(tmp, offsets[toKey(g)])
		indexId.Write(tmp)
	}

	// Create index sorted after name
	var indexName bytes.Buffer
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})
	for _, g := range sorted {
		le.PutUint64(tmp, offsets[toKey(g)])
		indexName.Write(tmp)
	}

	// Sanity check
	if len(grs)*8 != indexOrig.Len() ||
		indexOrig.Len() != indexId.Len() ||
		indexId.Len() != indexName.Len() {
		return fmt.Errorf("indexes have inconsistent length")
	}

	// Write result

	// magic
	w.Write([]byte("NSS-CASH"))
	// version
	le.PutUint64(tmp, GroupVersion)
	w.Write(tmp)
	// count
	le.PutUint64(tmp, uint64(len(grs)))
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
