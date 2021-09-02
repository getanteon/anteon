package configReader

import (
	"strings"

	"ddosify.com/hammer/core/types"
)

type ConfigReader interface {
	init(configPath string) error
	CreateHammer() (types.Hammer, error)
}

func NewConfigReader(path string, configType string) (reader ConfigReader, err error) {
	if strings.EqualFold(configType, "jsonReader") {
		reader = &jsonReader{}
	} 
	err = reader.init(path)

	return
}
