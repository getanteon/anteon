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
	"charm.land/bubbles/v2/progress"
	"charm.land/lipgloss/v2"
)

// styles carry emphasis only; every state they mark is also written as text or a symbol.
type styles struct {
	title, label, value, warn, bad, good, dim lipgloss.Style
}

func newStyles(noColor bool) styles {
	if noColor {
		p := lipgloss.NewStyle()
		return styles{p, p, p, p, p, p, p}
	}
	return styles{
		title: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7D56F4")),
		label: lipgloss.NewStyle().Foreground(lipgloss.Color("#8A8A8A")),
		value: lipgloss.NewStyle().Bold(true),
		warn:  lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E5A50A")),
		bad:   lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E01B24")),
		good:  lipgloss.NewStyle().Foreground(lipgloss.Color("#2EC27E")),
		dim:   lipgloss.NewStyle().Faint(true),
	}
}

func newBar(noColor bool) progress.Model {
	if noColor {
		return progress.New(
			progress.WithoutPercentage(),
			progress.WithFillCharacters('#', '-'),
			progress.WithColors(lipgloss.NoColor{}),
			func(m *progress.Model) { m.EmptyColor = lipgloss.NoColor{} },
		)
	}
	return progress.New(
		progress.WithoutPercentage(),
		progress.WithFillCharacters('█', '░'),
		progress.WithColors(lipgloss.Color("#7D56F4")),
	)
}
