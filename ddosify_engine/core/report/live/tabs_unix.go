//go:build darwin || linux || freebsd || netbsd || openbsd

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

package live

import (
	"golang.org/x/sys/unix"
)

// expandTabs sets the terminal to expand tabs (TAB3, OXTABS) while the TUI runs. Bubble Tea moves the
// cursor with hard tabs when the terminal does not, and a capture of the screen then holds tab
// characters that no longer line up. It returns the function that puts the setting back.
func expandTabs(fd uintptr) func() {
	t, err := unix.IoctlGetTermios(int(fd), ioctlGetTermios)
	if err != nil || t.Oflag&unix.TABDLY == unix.TAB3 {
		return func() {}
	}
	prev := *t
	t.Oflag = t.Oflag&^unix.TABDLY | unix.TAB3
	if err := unix.IoctlSetTermios(int(fd), ioctlSetTermios, t); err != nil {
		return func() {}
	}
	return func() { _ = unix.IoctlSetTermios(int(fd), ioctlSetTermios, &prev) }
}
