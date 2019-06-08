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

#include "cash.h"


struct file {
    int fd;
    size_t size;
    uint64_t next_index; // used by getpwent (pw.c)

    const struct header *header;
};

bool map_file(const char *path, struct file *f) __attribute__((visibility("hidden")));
void unmap_file(struct file *f) __attribute__((visibility("hidden")));

#endif
