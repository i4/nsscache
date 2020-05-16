/*
 * Tests for the NSS cash module
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

#include <assert.h>
#include <errno.h>
#include <limits.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <unistd.h>

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

static void test_limits(void) {
    char large_member[65525];
    memset(large_member, 'X', sizeof(large_member));
    large_member[sizeof(large_member)-1] = '\0';

    char many_members[54603]; // 5461 members
    memset(many_members, 'X', sizeof(many_members));
    for (int i = 9; i < (int)sizeof(many_members); i += 10) {
        many_members[i-1] = (char)('A' + i % ('Z' - 'A'));
        many_members[i] = ',';
    }
    many_members[sizeof(many_members)-1] = '\0';

    int r;
    FILE *fh;

    const char *nsscash_cmd = "../nsscash convert group "
        "tests/limits tests/limits.nsscash 2> /dev/null";

    // Entries which will not fit in uint16_t, nsscash must abort

    fh = fopen("tests/limits", "w");
    assert(fh != NULL);
    r = fprintf(fh, "test:x:42:A%s\n", large_member);
    assert(r == 65536);
    r = fclose(fh);
    assert(r == 0);

    r = system(nsscash_cmd);
    assert(r != -1);
    assert(WIFEXITED(r) && WEXITSTATUS(r) == 1);

    fh = fopen("tests/limits", "w");
    assert(fh != NULL);
    r = fprintf(fh, "many:x:4711:%s%s\n", many_members, many_members);
    assert(r == 109217);
    r = fclose(fh);
    assert(r == 0);

    r = system(nsscash_cmd);
    assert(r != -1);
    assert(WIFEXITED(r) && WEXITSTATUS(r) == 1);

    // Largest entries which will fit

    fh = fopen("tests/limits", "w");
    assert(fh != NULL);
    r = fprintf(fh, "test:x:42:%s\n", large_member);
    assert(r == 65535);
    r = fprintf(fh, "many:x:4711:%s\n", many_members);
    assert(r == 54615);
    r = fclose(fh);
    assert(r == 0);

    r = system(nsscash_cmd);
    assert(r != -1);
    assert(WIFEXITED(r) && WEXITSTATUS(r) == 0);

    r = rename("tests/group.nsscash", "tests/group.nsscash.tmp");
    assert(r == 0);
    r = rename("tests/limits.nsscash", "tests/group.nsscash");
    assert(r == 0);

    // Check if the entry can be retrieved

    struct group g;
    enum nss_status s;
    char tmp[sizeof(char **) + 1*sizeof(char *) + 1*sizeof(uint16_t) +
             4+1 + 1+1 + 65525 + 1];
    char tmp2[sizeof(char **) + 5462*sizeof(char *) + 5462*sizeof(uint16_t) +
             4+1 + 1+1 + 54603 + 1];
    int errnop = 0;

    s = _nss_cash_getgrgid_r(42, &g, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "test"));
    assert(g.gr_gid == 42);
    assert(g.gr_mem != NULL);
    assert(!strcmp(g.gr_mem[0], large_member));
    assert(g.gr_mem[1] == NULL);

    s = _nss_cash_getgrgid_r(4711, &g, tmp2, sizeof(tmp2), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(g.gr_name, "many"));
    assert(g.gr_gid == 4711);
    assert(g.gr_mem != NULL);
    for (int i = 0; i < 5461-1; i++) {
        char x[9+1];
        memset(x, 'X', sizeof(x));
        x[8] = (char)('A' + (i * 10 + 9) % ('Z' - 'A'));
        x[9] = '\0';
        assert(!strcmp(g.gr_mem[i], x));
    }
    assert(!strcmp(g.gr_mem[5461-1], "XX"));
    assert(g.gr_mem[5461] == NULL);

    r = rename("tests/group.nsscash.tmp", "tests/group.nsscash");
    assert(r == 0);

    r = unlink("tests/limits");
    assert(r == 0);
}

int main(void) {
    test_getgrent();
    test_getgrgid();
    test_getgrnam();

    test_limits();

    return EXIT_SUCCESS;
}
