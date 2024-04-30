package extraction

import (
	"bytes"
	"fmt"

	"github.com/antchfx/htmlquery"
)

type htmlExtractor struct {
}

func (xe htmlExtractor) extractFromByteSlice(source []byte, xPath string) (interface{}, error) {
	reader := bytes.NewBuffer(source)
	rootNode, err := htmlquery.Parse(reader)
	if err != nil {
		return nil, err
	}

	// returns the first matched element
	foundNode, err := htmlquery.Query(rootNode, xPath)
	if foundNode == nil || err != nil {
		return nil, fmt.Errorf("no match for the xPath_html: %s", xPath)
	}

	return foundNode.FirstChild.Data, nil
}

func (xe htmlExtractor) extractFromString(source string, xPath string) (interface{}, error) {
	reader := bytes.NewBufferString(source)
	rootNode, err := htmlquery.Parse(reader)
	if err != nil {
		return nil, err
	}

	// returns the first matched element
	foundNode, err := htmlquery.Query(rootNode, xPath)
	if foundNode == nil || err != nil {
		return nil, fmt.Errorf("no match for this xpath_html")
	}

	return foundNode.FirstChild.Data, nil
}
