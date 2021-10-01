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

package config

import (
	"fmt"
	"reflect"

	"go.ddosify.com/ddosify/core/types"
)

var AvailableConfigReader = make(map[string]ConfigReader)

// ConfigReader is the interface that abstracts different config reader implementations.
type ConfigReader interface {
	Init([]byte) error
	CreateHammer() (types.Hammer, error)
}

// NewConfigReader is the factory method of the ConfigReader.
func NewConfigReader(config []byte, configType string) (reader ConfigReader, err error) {
	if val, ok := AvailableConfigReader[configType]; ok {
		// Create a new object from the service type
		reader = reflect.New(reflect.TypeOf(val).Elem()).Interface().(ConfigReader)
		err = reader.Init(config)
	} else {
		err = fmt.Errorf("unsupported config reader type: %s", configType)
	}
	return
}
