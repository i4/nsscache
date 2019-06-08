/*
 * Tests for the NSS cash module
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

#include <assert.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <string.h>
#include <limits.h>

#include "../cash_nss.h"


static void test_getgrent(void) {
    struct group g;
    enum nss_status s;
    char tmp[1024];
    char tmp_small[10];
    int errnop = 0;

    // Test one setgrent/getgrent/endgrent round

    s = _nss_cash_setgrent(0);
    assert(s == NSS_STATUS_SUCCESS);

    // Multiple calls with too small buffer don't advance any internal indices
    s = _nss_cash_getgrent_r(&g, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);
    s = _nss_cash_getgrent_r(&g, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);
    s = _nss_cash_getgrent_r(&g, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);

    s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "root"));
    assert(!strcmp(g.gr_passwd, "x"));
    assert(g.gr_gid == 0);
    assert(g.gr_mem != NULL);
    assert(g.gr_mem[0] == NULL);

    s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "daemon"));
    assert(g.gr_gid == 1);
    assert(g.gr_mem != NULL);
    assert(!strcmp(g.gr_mem[0], "andariel"));
    assert(!strcmp(g.gr_mem[1], "duriel"));
    assert(!strcmp(g.gr_mem[2], "mephisto"));
    assert(!strcmp(g.gr_mem[3], "diablo"));
    assert(!strcmp(g.gr_mem[4], "baal"));
    assert(g.gr_mem[5] == NULL);
    for (int i = 0; i < 21; i++) {
        s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
        assert(s == NSS_STATUS_SUCCESS);
    }
    s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "www-data"));
    assert(!strcmp(g.gr_passwd, "x"));
    assert(g.gr_gid == 33);
    assert(g.gr_mem != NULL);
    assert(!strcmp(g.gr_mem[0], "nobody"));
    assert(g.gr_mem[1] == NULL);
    for (int i = 0; i < 29; i++) {
        s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
        assert(s == NSS_STATUS_SUCCESS);
    }
    s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "postfix"));
    s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "postdrop"));
    s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_NOTFOUND);
    assert(errnop == ENOENT);

    s = _nss_cash_endgrent();
    assert(s == NSS_STATUS_SUCCESS);


    // Test proper reset

    s = _nss_cash_setgrent(0);
    assert(s == NSS_STATUS_SUCCESS);

    s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "root"));
    assert(!strcmp(g.gr_passwd, "x"));
    assert(g.gr_gid == 0);
    assert(g.gr_mem != NULL);
    assert(g.gr_mem[0] == NULL);

    s = _nss_cash_endgrent();
    assert(s == NSS_STATUS_SUCCESS);


    // Test proper reset the 2nd

    s = _nss_cash_setgrent(0);
    assert(s == NSS_STATUS_SUCCESS);

    s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "root"));
    assert(!strcmp(g.gr_passwd, "x"));
    assert(g.gr_gid == 0);
    assert(g.gr_mem != NULL);
    assert(g.gr_mem[0] == NULL);

    s = _nss_cash_endgrent();
    assert(s == NSS_STATUS_SUCCESS);


    // Test many rounds to check for open file leaks
    for (int i = 0; i < 10000; i++) {
        s = _nss_cash_setgrent(0);
        assert(s == NSS_STATUS_SUCCESS);

        s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
        assert(s == NSS_STATUS_SUCCESS);
        assert(!strcmp(g.gr_name, "root"));

        s = _nss_cash_endgrent();
        assert(s == NSS_STATUS_SUCCESS);
    }


    // Test with cash file is not present

    assert(rename("tests/group.nsscash", "tests/group.nsscash.tmp") == 0);
    s = _nss_cash_setgrent(0);
    assert(s == NSS_STATUS_SUCCESS);
    s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_UNAVAIL);
    assert(errnop == ENOENT);
    s = _nss_cash_getgrent_r(&g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_UNAVAIL);
    assert(errnop == ENOENT);
    s = _nss_cash_endgrent();
    assert(s == NSS_STATUS_SUCCESS);
    assert(rename("tests/group.nsscash.tmp", "tests/group.nsscash") == 0);
}

static void test_getgrgid(void) {
    struct group g;
    enum nss_status s;
    char tmp[1024];
    char tmp_small[10];
    int errnop = 0;

    s = _nss_cash_getgrgid_r(0, &g, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);
    s = _nss_cash_getgrgid_r(14, &g, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_NOTFOUND); // 14 does not exist
    assert(errnop == ENOENT);
    s = _nss_cash_getgrgid_r(65534, &g, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);

    s = _nss_cash_getgrgid_r(0, &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "root"));
    assert(!strcmp(g.gr_passwd, "x"));
    assert(g.gr_gid == 0);
    assert(g.gr_mem != NULL);
    assert(g.gr_mem[0] == NULL);

    s = _nss_cash_getgrgid_r(1, &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "daemon"));
    assert(g.gr_gid == 1);
    assert(g.gr_mem != NULL);
    assert(!strcmp(g.gr_mem[0], "andariel"));
    assert(!strcmp(g.gr_mem[1], "duriel"));
    assert(!strcmp(g.gr_mem[2], "mephisto"));
    assert(!strcmp(g.gr_mem[3], "diablo"));
    assert(!strcmp(g.gr_mem[4], "baal"));
    assert(g.gr_mem[5] == NULL);

    s = _nss_cash_getgrgid_r(11, &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_NOTFOUND);
    assert(errnop == ENOENT);

    s = _nss_cash_getgrgid_r(103, &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "systemd-network"));
    assert(!strcmp(g.gr_passwd, "x"));
    assert(g.gr_gid == 103);
    assert(g.gr_mem != NULL);
    assert(g.gr_mem[0] == NULL);

    s = _nss_cash_getgrgid_r(107, &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "kvm"));
    assert(!strcmp(g.gr_passwd, "x"));
    assert(g.gr_gid == 107);
    assert(g.gr_mem != NULL);
    assert(g.gr_mem[0] == NULL);

    s = _nss_cash_getgrgid_r(65534, &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "nogroup"));
    assert(!strcmp(g.gr_passwd, "x"));
    assert(g.gr_gid == 65534);
    assert(g.gr_mem != NULL);
    assert(g.gr_mem[0] == NULL);

    s = _nss_cash_getgrgid_r(INT_MAX, &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_NOTFOUND);
    assert(errnop == ENOENT);


    // Test with cash file is not present

    assert(rename("tests/group.nsscash", "tests/group.nsscash.tmp") == 0);
    s = _nss_cash_getgrgid_r(0, &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_UNAVAIL);
    assert(errnop == ENOENT);
    s = _nss_cash_getgrgid_r(14, &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_UNAVAIL);
    assert(errnop == ENOENT);
    assert(rename("tests/group.nsscash.tmp", "tests/group.nsscash") == 0);
}

static void test_getgrnam(void) {
    struct group g;
    enum nss_status s;
    char tmp[1024];
    char tmp_small[10];
    int errnop = 0;

    s = _nss_cash_getgrnam_r("root", &g, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);
    s = _nss_cash_getgrnam_r("nope", &g, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_NOTFOUND); // does not exist
    assert(errnop == ENOENT);
    s = _nss_cash_getgrnam_r("nogroup", &g, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);

    s = _nss_cash_getgrnam_r("root", &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "root"));
    assert(!strcmp(g.gr_passwd, "x"));
    assert(g.gr_gid == 0);
    assert(g.gr_mem != NULL);
    assert(g.gr_mem[0] == NULL);

    s = _nss_cash_getgrnam_r("daemon", &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "daemon"));
    assert(g.gr_gid == 1);
    assert(g.gr_mem != NULL);
    assert(!strcmp(g.gr_mem[0], "andariel"));
    assert(!strcmp(g.gr_mem[1], "duriel"));
    assert(!strcmp(g.gr_mem[2], "mephisto"));
    assert(!strcmp(g.gr_mem[3], "diablo"));
    assert(!strcmp(g.gr_mem[4], "baal"));
    assert(g.gr_mem[5] == NULL);

    s = _nss_cash_getgrnam_r("nope2", &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_NOTFOUND);
    assert(errnop == ENOENT);

    s = _nss_cash_getgrnam_r("systemd-network", &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "systemd-network"));
    assert(!strcmp(g.gr_passwd, "x"));
    assert(g.gr_gid == 103);
    assert(g.gr_mem != NULL);
    assert(g.gr_mem[0] == NULL);

    s = _nss_cash_getgrnam_r("postfix", &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "postfix"));
    assert(!strcmp(g.gr_passwd, "x"));
    assert(g.gr_gid == 114);
    assert(g.gr_mem != NULL);
    assert(g.gr_mem[0] == NULL);

    s = _nss_cash_getgrnam_r("", &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_NOTFOUND);
    assert(errnop == ENOENT);


    // Test with cash file is not present

    assert(rename("tests/group.nsscash", "tests/group.nsscash.tmp") == 0);
    s = _nss_cash_getgrnam_r("root", &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_UNAVAIL);
    assert(errnop == ENOENT);
    s = _nss_cash_getgrnam_r("nope", &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_UNAVAIL);
    assert(errnop == ENOENT);
    assert(rename("tests/group.nsscash.tmp", "tests/group.nsscash") == 0);
}

int main(void) {
    test_getgrent();
    test_getgrgid();
    test_getgrnam();

    return EXIT_SUCCESS;
}
