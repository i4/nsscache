/*
 * Load and unload nsscash files (header)
 *
 * Copyright (C) 2019  Simon Ruderich
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

#ifndef FILE_H
#define FILE_H

#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>


// Magic value at the beginning of each nsscash file (8 byte, without the
// trailing NUL)
#define MAGIC "NSS-CASH"

// Defined in Makefile
#ifndef NSSCASH_PASSWD_FILE
# define NSSCASH_PASSWD_FILE "/etc/passwd.nsscash"
#endif
#ifndef NSSCASH_GROUP_FILE
# define NSSCASH_GROUP_FILE "/etc/group.nsscash"
#endif


// header describes the on-disk (and, after loading via mmap, in-memory)
// structure of nsscash files.
struct header {
    char magic[8]; // magic string
    uint64_t version; // also doubles as byte-order check

    uint64_t count; // number of entries in this file

    // All offsets are relative to data
    uint64_t off_orig_index;
    uint64_t off_id_index;
    uint64_t off_name_index;
    uint64_t off_data;

    char data[];
} __attribute__((packed));

// file represents an open nsscash file.
struct file {
    int fd;
    size_t size;

    const struct header *header;
    uint64_t next_index; // used by getpwent (pw.c)
};

bool map_file(const char *path, struct file *f) __attribute__((visibility("hidden")));
void unmap_file(struct file *f) __attribute__((visibility("hidden")));

#endif
