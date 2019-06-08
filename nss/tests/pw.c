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


static void test_getpwent(void) {
    struct passwd p;
    enum nss_status s;
    char tmp[1024];
    char tmp_small[10];
    int errnop = 0;

    // Test one setpwent/getpwent/endpwent round

    s = _nss_cash_setpwent(0);
    assert(s == NSS_STATUS_SUCCESS);

    // Multiple calls with too small buffer don't advance any internal indices
    s = _nss_cash_getpwent_r(&p, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);
    s = _nss_cash_getpwent_r(&p, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);
    s = _nss_cash_getpwent_r(&p, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);

    s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "root"));
    assert(!strcmp(p.pw_passwd, "x"));
    assert(p.pw_uid == 0);
    assert(p.pw_gid == 0);
    assert(!strcmp(p.pw_gecos, "root"));
    assert(!strcmp(p.pw_dir, "/root"));
    assert(!strcmp(p.pw_shell, "/bin/bash"));

    s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "daemon"));
    for (int i = 0; i < 10; i++) {
        s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
        assert(s == NSS_STATUS_SUCCESS);
    }
    s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "www-data"));
    for (int i = 0; i < 12; i++) {
        s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
        assert(s == NSS_STATUS_SUCCESS);
    }
    s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "_rpc"));
    s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "postfix"));
    s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_NOTFOUND);
    assert(errnop == ENOENT);

    s = _nss_cash_endpwent();
    assert(s == NSS_STATUS_SUCCESS);


    // Test proper reset

    s = _nss_cash_setpwent(0);
    assert(s == NSS_STATUS_SUCCESS);

    s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "root"));
    assert(!strcmp(p.pw_passwd, "x"));
    assert(p.pw_uid == 0);
    assert(p.pw_gid == 0);
    assert(!strcmp(p.pw_gecos, "root"));
    assert(!strcmp(p.pw_dir, "/root"));
    assert(!strcmp(p.pw_shell, "/bin/bash"));

    s = _nss_cash_endpwent();
    assert(s == NSS_STATUS_SUCCESS);


    // Test proper reset the 2nd

    s = _nss_cash_setpwent(0);
    assert(s == NSS_STATUS_SUCCESS);

    s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "root"));
    assert(!strcmp(p.pw_passwd, "x"));
    assert(p.pw_uid == 0);
    assert(p.pw_gid == 0);
    assert(!strcmp(p.pw_gecos, "root"));
    assert(!strcmp(p.pw_dir, "/root"));
    assert(!strcmp(p.pw_shell, "/bin/bash"));

    s = _nss_cash_endpwent();
    assert(s == NSS_STATUS_SUCCESS);


    // Test many rounds to check for open file leaks
    for (int i = 0; i < 10000; i++) {
        s = _nss_cash_setpwent(0);
        assert(s == NSS_STATUS_SUCCESS);

        s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
        assert(s == NSS_STATUS_SUCCESS);
        assert(!strcmp(p.pw_name, "root"));

        s = _nss_cash_endpwent();
        assert(s == NSS_STATUS_SUCCESS);
    }


    // Test with cash file is not present

    assert(rename("tests/passwd.nsscash", "tests/passwd.nsscash.tmp") == 0);
    s = _nss_cash_setpwent(0);
    assert(s == NSS_STATUS_SUCCESS);
    s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_UNAVAIL);
    assert(errnop == ENOENT);
    s = _nss_cash_getpwent_r(&p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_UNAVAIL);
    assert(errnop == ENOENT);
    s = _nss_cash_endpwent();
    assert(s == NSS_STATUS_SUCCESS);
    assert(rename("tests/passwd.nsscash.tmp", "tests/passwd.nsscash") == 0);
}

static void test_getpwuid(void) {
    struct passwd p;
    enum nss_status s;
    char tmp[1024];
    char tmp_small[10];
    int errnop = 0;

    s = _nss_cash_getpwuid_r(0, &p, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);
    s = _nss_cash_getpwuid_r(42, &p, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_NOTFOUND); // 42 does not exist
    assert(errnop == ENOENT);
    s = _nss_cash_getpwuid_r(65534, &p, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);

    s = _nss_cash_getpwuid_r(0, &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "root"));
    assert(!strcmp(p.pw_passwd, "x"));
    assert(p.pw_uid == 0);
    assert(p.pw_gid == 0);
    assert(!strcmp(p.pw_gecos, "root"));
    assert(!strcmp(p.pw_dir, "/root"));
    assert(!strcmp(p.pw_shell, "/bin/bash"));

    s = _nss_cash_getpwuid_r(1, &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "daemon"));
    assert(!strcmp(p.pw_passwd, "x"));
    assert(p.pw_uid == 1);
    assert(p.pw_gid == 1);
    assert(!strcmp(p.pw_gecos, "daemon"));
    assert(!strcmp(p.pw_dir, "/usr/sbin"));
    assert(!strcmp(p.pw_shell, "/usr/sbin/nologin"));

    s = _nss_cash_getpwuid_r(11, &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_NOTFOUND);
    assert(errnop == ENOENT);

    s = _nss_cash_getpwuid_r(102, &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "systemd-network"));
    assert(!strcmp(p.pw_passwd, "x"));
    assert(p.pw_uid == 102);
    assert(p.pw_gid == 103);
    assert(!strcmp(p.pw_gecos, "systemd Network Management,,,"));
    assert(!strcmp(p.pw_dir, "/run/systemd"));
    assert(!strcmp(p.pw_shell, "/usr/sbin/nologin"));

    s = _nss_cash_getpwuid_r(107, &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "postfix"));
    assert(!strcmp(p.pw_passwd, "x"));
    assert(p.pw_uid == 107);
    assert(p.pw_gid == 114);
    assert(!strcmp(p.pw_gecos, ""));
    assert(!strcmp(p.pw_dir, "/var/spool/postfix"));
    assert(!strcmp(p.pw_shell, "/usr/sbin/nologin"));

    s = _nss_cash_getpwuid_r(INT_MAX, &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_NOTFOUND);
    assert(errnop == ENOENT);


    // Test with cash file is not present

    assert(rename("tests/passwd.nsscash", "tests/passwd.nsscash.tmp") == 0);
    s = _nss_cash_getpwuid_r(0, &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_UNAVAIL);
    assert(errnop == ENOENT);
    s = _nss_cash_getpwuid_r(42, &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_UNAVAIL);
    assert(errnop == ENOENT);
    assert(rename("tests/passwd.nsscash.tmp", "tests/passwd.nsscash") == 0);
}

static void test_getpwnam(void) {
    struct passwd p;
    enum nss_status s;
    char tmp[1024];
    char tmp_small[10];
    int errnop = 0;

    s = _nss_cash_getpwnam_r("root", &p, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);
    s = _nss_cash_getpwnam_r("nope", &p, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_NOTFOUND); // does not exist
    assert(errnop == ENOENT);
    s = _nss_cash_getpwnam_r("nobody", &p, tmp_small, sizeof(tmp_small), &errnop);
    assert(s == NSS_STATUS_TRYAGAIN);
    assert(errnop == ERANGE);

    s = _nss_cash_getpwnam_r("root", &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "root"));
    assert(!strcmp(p.pw_passwd, "x"));
    assert(p.pw_uid == 0);
    assert(p.pw_gid == 0);
    assert(!strcmp(p.pw_gecos, "root"));
    assert(!strcmp(p.pw_dir, "/root"));
    assert(!strcmp(p.pw_shell, "/bin/bash"));

    s = _nss_cash_getpwnam_r("daemon", &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "daemon"));
    assert(!strcmp(p.pw_passwd, "x"));
    assert(p.pw_uid == 1);
    assert(p.pw_gid == 1);
    assert(!strcmp(p.pw_gecos, "daemon"));
    assert(!strcmp(p.pw_dir, "/usr/sbin"));
    assert(!strcmp(p.pw_shell, "/usr/sbin/nologin"));

    s = _nss_cash_getpwnam_r("nope2", &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_NOTFOUND);
    assert(errnop == ENOENT);

    s = _nss_cash_getpwnam_r("systemd-network", &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "systemd-network"));
    assert(!strcmp(p.pw_passwd, "x"));
    assert(p.pw_uid == 102);
    assert(p.pw_gid == 103);
    assert(!strcmp(p.pw_gecos, "systemd Network Management,,,"));
    assert(!strcmp(p.pw_dir, "/run/systemd"));
    assert(!strcmp(p.pw_shell, "/usr/sbin/nologin"));

    s = _nss_cash_getpwnam_r("postfix", &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_SUCCESS);
    assert(!strcmp(p.pw_name, "postfix"));
    assert(!strcmp(p.pw_passwd, "x"));
    assert(p.pw_uid == 107);
    assert(p.pw_gid == 114);
    assert(!strcmp(p.pw_gecos, ""));
    assert(!strcmp(p.pw_dir, "/var/spool/postfix"));
    assert(!strcmp(p.pw_shell, "/usr/sbin/nologin"));

    s = _nss_cash_getpwnam_r("", &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_NOTFOUND);
    assert(errnop == ENOENT);


    // Test with cash file is not present

    assert(rename("tests/passwd.nsscash", "tests/passwd.nsscash.tmp") == 0);
    s = _nss_cash_getpwnam_r("root", &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_UNAVAIL);
    assert(errnop == ENOENT);
    s = _nss_cash_getpwnam_r("nope", &p, tmp, sizeof(tmp), &errnop);
    assert(s == NSS_STATUS_UNAVAIL);
    assert(errnop == ENOENT);
    assert(rename("tests/passwd.nsscash.tmp", "tests/passwd.nsscash") == 0);
}

int main(void) {
    test_getpwent();
    test_getpwuid();
    test_getpwnam();

    return EXIT_SUCCESS;
}
