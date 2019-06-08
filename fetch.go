// Download files via HTTP with support for If-Modified-Since

// Copyright (C) 2019  Simon Ruderich
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"io/ioutil"
	"net/http"
	"time"
)

// Global variable to permit reuse of connections (keep-alive)
var client *http.Client

func init() {
	client = &http.Client{}
}

func fetchIfModified(url string, lastModified *time.Time) (int, []byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, nil, err
	}
	if !lastModified.IsZero() {
		req.Header.Add("If-Modified-Since",
			lastModified.Format(http.TimeFormat))
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}

	modified, err := http.ParseTime(resp.Header.Get("Last-Modified"))
	if err == nil {
		*lastModified = modified
	}

	return resp.StatusCode, body, nil
}
