/*
 * Handle passwd entries via struct passwd
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

#include <errno.h>
#include <stddef.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <pthread.h>

#include "cash_nss.h"
#include "file.h"
#include "search.h"


// NOTE: This file is very similar to gr.c, keep in sync!

struct passwd_entry {
    uint64_t uid;
    uint64_t gid;

    //       off_name = 0, not stored on disk
    uint16_t off_passwd;
    uint16_t off_gecos;
    uint16_t off_dir;
    uint16_t off_shell;

    /*
     * Data contains all strings (name, passwd, gecos, dir, shell)
     * concatenated, with their trailing NUL. The off_* variables point to
     * beginning of each string.
     */
    uint16_t data_size; // size of data in bytes
    const char data[];
} __attribute__((packed));

static bool entry_to_passwd(const struct passwd_entry *e, struct passwd *p, char *tmp, size_t space) {
    if (space < e->data_size) {
        return false;
    }

    memcpy(tmp, e->data, e->data_size);
    p->pw_uid = (uid_t)e->uid;
    p->pw_gid = (gid_t)e->gid;
    p->pw_name = tmp + 0;
    p->pw_passwd = tmp + e->off_passwd;
    p->pw_gecos = tmp + e->off_gecos;
    p->pw_dir = tmp + e->off_dir;
    p->pw_shell = tmp + e->off_shell;

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

enum nss_status _nss_cash_setpwent(int x) {
    (void)x;

    // Unmap is necessary to detect changes when the file was replaced on
    // disk; getpwent_r will open the file if necessary when called
    internal_unmap_static_file();
    return NSS_STATUS_SUCCESS;
}

enum nss_status _nss_cash_endpwent(void) {
    internal_unmap_static_file();
    return NSS_STATUS_SUCCESS;
}

static enum nss_status internal_getpwent_r(struct passwd *result, char *buffer, size_t buflen) {
    // First call to getpwent_r, load file from disk
    if (static_file.header == NULL) {
        if (!map_file(NSSCASH_PASSWD_FILE, &static_file)) {
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
    if (!entry_to_passwd((struct passwd_entry *)e, result, buffer, buflen)) {
        errno = ERANGE;
        return NSS_STATUS_TRYAGAIN;
    }
    static_file.next_index++;

    return NSS_STATUS_SUCCESS;
}
enum nss_status _nss_cash_getpwent_r(struct passwd *result, char *buffer, size_t buflen, int *errnop) {
    pthread_mutex_lock(&static_file_lock);
    enum nss_status s = internal_getpwent_r(result, buffer, buflen);
    pthread_mutex_unlock(&static_file_lock);
    if (s != NSS_STATUS_SUCCESS) {
        *errnop = errno;
    }
    return s;
}


static enum nss_status internal_getpw(struct search_key *key, struct passwd *result, char *buffer, size_t buflen, int *errnop) {
    struct file f;
    if (!map_file(NSSCASH_PASSWD_FILE, &f)) {
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
    if (!entry_to_passwd((struct passwd_entry *)e, result, buffer, buflen)) {
        unmap_file(&f);
        errno = ERANGE;
        *errnop = errno;
        return NSS_STATUS_TRYAGAIN;
    }

    unmap_file(&f);
    return NSS_STATUS_SUCCESS;
}

enum nss_status _nss_cash_getpwuid_r(uid_t uid, struct passwd *result, char *buffer, size_t buflen, int *errnop) {
    struct search_key key = {
        .id = (uint64_t)uid,
        .offset = offsetof(struct passwd_entry, uid),
    };
    return internal_getpw(&key, result, buffer, buflen, errnop);
}

enum nss_status _nss_cash_getpwnam_r(const char *name, struct passwd *result, char *buffer, size_t buflen, int *errnop) {
    struct search_key key = {
        .name = name,
        .offset = sizeof(struct passwd_entry), // name is first value in data[]
    };
    return internal_getpw(&key, result, buffer, buflen, errnop);
}
