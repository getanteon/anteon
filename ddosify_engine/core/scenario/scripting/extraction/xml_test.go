package extraction

import (
	"fmt"
	"strings"
	"testing"
)

func TestXmlExtraction(t *testing.T) {
	expected := "XML Title"
	xmlSource := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" ?>
	<rss version="2.0">
	<channel>
	  <item>
		<title>%s</title>
	  </item>
	</channel>
	</rss>`, expected)

	xe := xmlExtractor{}
	xpath := "//item/title"
	val, err := xe.extractFromByteSlice([]byte(xmlSource), xpath)

	if err != nil {
		t.Errorf("TestXmlExtraction %v", err)
	}

	if !strings.EqualFold(val.(string), expected) {
		t.Errorf("TestXmlExtraction expected: %s, got: %s", expected, val)
	}
}

func TestXmlExtractionString(t *testing.T) {
	expected := "XML Title"
	xmlSource := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" ?>
	<rss version="2.0">
	<channel>
	  <item>
		<title>%s</title>
	  </item>
	</channel>
	</rss>`, expected)

	xe := xmlExtractor{}
	xpath := "//item/title"
	val, err := xe.extractFromString(xmlSource, xpath)

	if err != nil {
		t.Errorf("TestXmlExtraction %v", err)
	}

	if !strings.EqualFold(val.(string), expected) {
		t.Errorf("TestXmlExtraction expected: %s, got: %s", expected, val)
	}
}

func TestXmlExtraction_PathNotFound(t *testing.T) {
	expected := "XML Title"
	xmlSource := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8" ?>
	<rss version="2.0">
	<channel>
	  <item>
		<title>%s</title>
	  </item>
	</channel>
	</rss>`, expected)

	xe := xmlExtractor{}
	xpath := "//item3/title"
	_, err := xe.extractFromByteSlice([]byte(xmlSource), xpath)

	if err == nil {
		t.Errorf("TestXmlExtraction_PathNotFound, should be err, got :%v", err)
	}
}

func TestInvalidXml(t *testing.T) {
	xmlSource := `invalid xml source`

	xe := xmlExtractor{}
	xpath := "//item3/title"
	_, err := xe.extractFromByteSlice([]byte(xmlSource), xpath)

	if err == nil {
		t.Errorf("TestInvalidXml, should be err, got :%v", err)
	}
}
