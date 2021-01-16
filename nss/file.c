/*
 * Load and unload nsscash files
 *
 * Copyright (C) 2019-2021  Simon Ruderich
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

#include "file.h"

#include <errno.h>
#include <fcntl.h>
#include <stdlib.h>
#include <string.h>
#include <sys/mman.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <unistd.h>


bool map_file(const char *path, struct file *f) {
    // Fully initialize the struct for unmap_file() and other users
    memset(f, 0, sizeof(*f));

    f->fd = open(path, O_RDONLY | O_CLOEXEC);
    if (f->fd < 0) {
        goto fail;
    }
    struct stat s;
    if (fstat(f->fd, &s)) {
        goto fail;
    }
    f->size = (size_t)s.st_size; // for munmap()

    // mmap is used for speed and simple random access
    void *x = mmap(NULL, f->size, PROT_READ, MAP_PRIVATE, f->fd, 0);
    if (x == MAP_FAILED) {
        goto fail;
    }

    const struct header *h = x;
    f->header = h;

    // Check MAGIC
    if (memcmp(h->magic, MAGIC, sizeof(h->magic))) {
        errno = EINVAL;
        goto fail;
    }
    // Only version 1 is supported at the moment; this will also prevent
    // running on big-endian systems which is currently not possible
    if (h->version != 1) {
        errno = EINVAL;
        goto fail;
    }

    return true;

fail: {
        int save_errno = errno;
        unmap_file(f);
        errno = save_errno;
        return false;
    }
}

void unmap_file(struct file *f) {
    if (f->header != NULL) {
        munmap((void *)f->header, f->size);
        f->header = NULL;
    }
    if (f->fd >= 0) {
        close(f->fd);
        f->fd = -1;
    }
}
