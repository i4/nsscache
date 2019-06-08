/*
 * NSS function definitions provided by this NSS module
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

#ifndef CASH_NSS_H
#define CASH_NSS_H

#include <nss.h>
#include <pwd.h>


enum nss_status _nss_cash_setpwent(int);
enum nss_status _nss_cash_endpwent(void);
enum nss_status _nss_cash_getpwent_r(struct passwd *result, char *buffer, size_t buflen, int *errnop);
enum nss_status _nss_cash_getpwuid_r(uid_t uid, struct passwd *result, char *buffer, size_t buflen, int *errnop);
enum nss_status _nss_cash_getpwnam_r(const char *name, struct passwd *result, char *buffer, size_t buflen, int *errnop);

#endif
