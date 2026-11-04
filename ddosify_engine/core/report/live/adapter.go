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

// Adapt converts the engine's own snapshot type into Snapshot as it arrives.
// The engine's type is defined on v2/foundation; conv is the one function to write once it lands.
// The returned channel closes when in closes.
func Adapt[T any](in <-chan T, conv func(T) Snapshot) <-chan Snapshot {
	out := make(chan Snapshot, 4)
	go func() {
		defer close(out)
		for v := range in {
			out <- conv(v)
		}
	}()
	return out
}
