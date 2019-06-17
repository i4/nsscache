/*
 * Search entries in nsscash files by using indices and binary search
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

#include "search.h"

#include <stdlib.h>
#include <string.h>


static int bsearch_callback(const void *x, const void *y) {
    const struct search_key *key = x;

    uint64_t offset = *(const uint64_t *)y; // from index
    const void *member = (const char *)key->data + offset + key->offset;

    // Lookup by name (char *)
    if (key->name != NULL) {
        const char *name = member;
        return strcmp(key->name, name);

    // Lookup by ID (uint64_t)
    } else {
        const uint64_t *id = member;
        if (key->id < *id) {
            return -1;
        } else if (key->id == *id) {
            return 0;
        } else {
            return +1;
        }
    }
}

// search performs a binary search on an index, described by key and index.
uint64_t *search(const struct search_key *key, const void *index, uint64_t count) {
    return bsearch(key, index, count, sizeof(uint64_t), bsearch_callback);
}
