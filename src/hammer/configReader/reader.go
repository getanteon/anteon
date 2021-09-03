package configReader

import (
	"io/ioutil"
	"os"
	"strings"

	"ddosify.com/hammer/core/types"
)

type ConfigReader interface {
	init([]byte) error
	CreateHammer() (types.Hammer, error)
}

func NewConfigReaderFromFile(path string, configType string) (reader ConfigReader, err error) {
	if strings.EqualFold(configType, "jsonReader") {
		reader = &jsonReader{}
	} 

	jsonFile, err := os.Open(path)
	if err != nil {
		return
	}

	byteValue, _ := ioutil.ReadAll(jsonFile)
	if err != nil {
		return
	}
	err = reader.init(byteValue)

	return
}
