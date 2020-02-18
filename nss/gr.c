/*
 * Handle group entries via struct group
 *
 * Copyright (C) 2019-2020  Simon Ruderich
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

#include <errno.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <pthread.h>

#include "cash_nss.h"
#include "file.h"
#include "search.h"


// NOTE: This file is very similar to pw.c, keep in sync!

// TODO: adapt offsets to 32 bit to fit more than 5000 users per group (for 9
// byte user names)
struct group_entry {
    uint64_t gid;

    //       off_name = 0, not stored on disk
    uint16_t off_passwd;
    uint16_t off_mem_off;

    uint16_t mem_count; // group member count

    /*
     * Data contains all strings (name, passwd) concatenated, with their
     * trailing NUL. The off_* variables point to beginning of each string.
     *
     * After that the offsets of the members of the group are stored as
     * mem_count uint16_t values, followed by the member names concatenated as
     * with the strings above.
     *
     * All offsets are relative to the beginning of data.
     */
    uint16_t data_size; // size of data in bytes
    const char data[];
} __attribute__((packed));

static bool entry_to_group(const struct group_entry *e, struct group *g, char *tmp, size_t space) {
    // Space required for the gr_mem array
    const size_t mem_size = (size_t)(e->mem_count + 1) * sizeof(char *);

    if (space < e->data_size + mem_size) {
        return false;
    }

    char **groups = (char **)tmp;

    const uint16_t *offs_mem = (const uint16_t *)(e->data + e->off_mem_off);
    for (uint16_t i = 0; i < e->mem_count; i++) {
        groups[i] = tmp + mem_size + offs_mem[i];
    }
    groups[e->mem_count] = NULL;

    // This unnecessarily copies offs_mem[] as well but keeps the code simpler
    // and the meaning of variables consistent with pw.c
    memcpy(tmp + mem_size, e->data, e->data_size);

    g->gr_gid = (gid_t)e->gid;
    g->gr_name = tmp + mem_size + 0;
    g->gr_passwd = tmp + mem_size + e->off_passwd;
    g->gr_mem = groups;

    return true;
}


static struct file static_file = {
    .fd = -1,
};
static pthread_mutex_t static_file_lock = PTHREAD_MUTEX_INITIALIZER;

static void internal_unmap_static_file(void) {
    pthread_mutex_lock(&static_file_lock);
    unmap_file(&static_file);
    pthread_mutex_unlock(&static_file_lock);
}

enum nss_status _nss_cash_setgrent(int x) {
    (void)x;

    // Unmap is necessary to detect changes when the file was replaced on
    // disk; getgrent_r will open the file if necessary when called
    internal_unmap_static_file();
    return NSS_STATUS_SUCCESS;
}

enum nss_status _nss_cash_endgrent(void) {
    internal_unmap_static_file();
    return NSS_STATUS_SUCCESS;
}

static enum nss_status internal_getgrent_r(struct group *result, char *buffer, size_t buflen) {
    // First call to getgrent_r, load file from disk
    if (static_file.header == NULL) {
        if (!map_file(NSSCASH_GROUP_FILE, &static_file)) {
            return NSS_STATUS_UNAVAIL;
        }
    }

    const struct header *h = static_file.header;
    // End of "file", stop
    if (static_file.next_index >= h->count) {
        errno = ENOENT;
        return NSS_STATUS_NOTFOUND;
    }

    uint64_t *off_orig = (uint64_t *)(h->data + h->off_orig_index);
    const char *e = h->data + h->off_data + off_orig[static_file.next_index];
    if (!entry_to_group((struct group_entry *)e, result, buffer, buflen)) {
        errno = ERANGE;
        return NSS_STATUS_TRYAGAIN;
    }
    static_file.next_index++;

    return NSS_STATUS_SUCCESS;
}
enum nss_status _nss_cash_getgrent_r(struct group *result, char *buffer, size_t buflen, int *errnop) {
    pthread_mutex_lock(&static_file_lock);
    enum nss_status s = internal_getgrent_r(result, buffer, buflen);
    pthread_mutex_unlock(&static_file_lock);
    if (s != NSS_STATUS_SUCCESS) {
        *errnop = errno;
    }
    return s;
}


static enum nss_status internal_getgr(struct search_key *key, struct group *result, char *buffer, size_t buflen, int *errnop) {
    struct file f;
    if (!map_file(NSSCASH_GROUP_FILE, &f)) {
        *errnop = errno;
        return NSS_STATUS_UNAVAIL;
    }
    const struct header *h = f.header;

    key->data = h->data + h->off_data;
    uint64_t off_index = (key->name != NULL)
                       ? h->off_name_index
                       : h->off_id_index;
    uint64_t *off = search(key, h->data + off_index, h->count);
    if (off == NULL) {
        unmap_file(&f);
        errno = ENOENT;
        *errnop = errno;
        return NSS_STATUS_NOTFOUND;
    }

    const char *e = key->data + *off;
    if (!entry_to_group((struct group_entry *)e, result, buffer, buflen)) {
        unmap_file(&f);
        errno = ERANGE;
        *errnop = errno;
        return NSS_STATUS_TRYAGAIN;
    }

    unmap_file(&f);
    return NSS_STATUS_SUCCESS;
}

enum nss_status _nss_cash_getgrgid_r(gid_t gid, struct group *result, char *buffer, size_t buflen, int *errnop) {
    struct search_key key = {
        .id = (uint64_t)gid,
        .offset = offsetof(struct group_entry, gid),
    };
    return internal_getgr(&key, result, buffer, buflen, errnop);
}

enum nss_status _nss_cash_getgrnam_r(const char *name, struct group *result, char *buffer, size_t buflen, int *errnop) {
    struct search_key key = {
        .name = name,
        .offset = sizeof(struct group_entry), // name is first value in data[]
    };
    return internal_getgr(&key, result, buffer, buflen, errnop);
}
