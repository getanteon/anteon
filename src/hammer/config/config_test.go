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
	"reflect"
	"testing"
)

func TestNewConfigReader(t *testing.T) {
	t.Parallel()
	config_path := "config_testdata/config.json"
	reader, err := NewConfigReader(config_path, ConfigTypeJson)

	if err != nil {
		t.Errorf("TestNewConfigReader errored: %v", err)
	}

	if reflect.TypeOf(reader) != reflect.TypeOf(&jsonReader{}) {
		t.Errorf("Expected jsonReader found: %v", reflect.TypeOf(reader))
	}
}

func TestNewConfigReaderInvalidConfigType(t *testing.T) {
	t.Parallel()
	config_path := "config_testdata/config.json"
	_, err := NewConfigReader(config_path, "invalidConfigType")

	if err == nil {
		t.Errorf("TestNewConfigReaderInvalidConfigType errored")
	}

}

func TestNewConfigReaderInvalidFilePath(t *testing.T) {
	t.Parallel()
	config_path := "config_testdata/invalid_file_path.json"
	_, err := NewConfigReader(config_path, ConfigTypeJson)

	if err == nil {
		t.Errorf("TestNewConfigReaderInvalidFilePath errored")
	}
}

func TestNewConfigReaderIncorrectJsonFile(t *testing.T) {
	t.Parallel()
	config_path := "config_testdata/incorrect_formatted.json"
	_, err := NewConfigReader(config_path, ConfigTypeJson)

	if err == nil {
		t.Errorf("TestNewConfigReaderInvalidFilePath errored")
	}
}
