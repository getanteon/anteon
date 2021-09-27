/*
*
*	Ddosify - Load testing tool for any web system.
*   Copyright (C) 2021  Ddosify (https://ddosify.com)
*
*   This program is free software: you can redistribute it and/or modify
*   it under the terms of the GNU Affero General Public License as published
*   by the Free Software Foundation, either version 3 of the License, or
*   (at your option) any later version.
*
*   This program is distributed in the hope that it will be useful,
*   but WITHOUT ANY WARRANTY; without even the implied warranty of
*   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
*   GNU Affero General Public License for more details.
*
*   You should have received a copy of the GNU Affero General Public License
*   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*
 */

package util

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	validator "github.com/asaskevich/govalidator"
)

// Checks if the given string is in the given list of strings
func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// Checks if the system is running for tests.
func IsSystemInTestMode() bool {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			return true
		}
	}
	return false
}

// Converts the given str to *url.URL.
// If the str does not have a Scheme segment then defaultProtocol parameter is used as a Scheme.
// Returns error if the given str is not a URL.
func StrToURL(defaultProtocol string, str string) (*url.URL, error) {
	u, err := url.Parse(str)
	if err != nil {
		return nil, fmt.Errorf("invalid target url")
	}

	// Without protocol, url.Parse returns Host empty and pass the whole value to Path.
	// If the protocol empty we should add default scheme then create new URL then check again.
	if u.Scheme == "" {
		u.Scheme = strings.ToLower(defaultProtocol)
		if !validator.IsURL(u.String()) {
			return nil, fmt.Errorf("invalid target url")
		}
	}

	return u, nil
}
