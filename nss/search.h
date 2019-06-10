/*
 * Search entries in nsscash files by using indices and binary search (header)
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

#ifndef SEARCH_H
#define SEARCH_H

#include <stdint.h>


struct search_key {
    const char *name;
    const uint64_t *id;

    const void *data;
    uint64_t offset;
};

uint64_t *search(struct search_key *key, const void *index, uint64_t count) __attribute__((visibility("hidden")));

#endif
