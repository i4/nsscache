/*
 * General header of nsscash
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

#ifndef CASH_H
#define CASH_H

#include <stdint.h>


// Global constants

#define MAGIC "NSS-CASH"

// Defined in Makefile
#ifndef NSSCASH_PASSWD_FILE
# define NSSCASH_PASSWD_FILE "/etc/passwd.nsscash"
#endif


// Global structs

struct header {
    char magic[8]; // magic string
    uint64_t version; // also doubles as byte-order check

    uint64_t count;

    // All offsets are relative to data
    uint64_t off_orig_index;
    uint64_t off_id_index;
    uint64_t off_name_index;
    uint64_t off_data;

    char data[];
} __attribute__((packed));

#endif
