package extraction

import (
	"fmt"
	"strings"
	"testing"
)

func TestHtmlExtraction(t *testing.T) {
	expected := "Html Title"
	HtmlSource := fmt.Sprintf(`<!DOCTYPE html>
	<html>
	<body>
	<h1>%s</h1>
	<p>My first paragraph.</p>
	</body>
	</html>`, expected)

	xe := htmlExtractor{}
	xpath := "//body/h1"
	val, err := xe.extractFromByteSlice([]byte(HtmlSource), xpath)

	if err != nil {
		t.Errorf("TestHtmlExtraction %v", err)
	}

	if !strings.EqualFold(val.(string), expected) {
		t.Errorf("TestHtmlExtraction expected: %s, got: %s", expected, val)
	}
}

func TestHtmlExtractionSeveralNode(t *testing.T) {
	//should extract only the first one
	expected := "Html Title"
	HtmlSource := fmt.Sprintf(`<!DOCTYPE html>
	<html>
	<body>
	<h1>%s</h1>
	<h1>another node</h1>
	<p>My first paragraph.</p>
	</body>
	</html>`, expected)

	xe := htmlExtractor{}
	xpath := "//h1"
	val, err := xe.extractFromByteSlice([]byte(HtmlSource), xpath)

	if err != nil {
		t.Errorf("TestHtmlExtraction %v", err)
	}

	if !strings.EqualFold(val.(string), expected) {
		t.Errorf("TestHtmlExtraction expected: %s, got: %s", expected, val)
	}
}

func TestHtmlExtraction_PathNotFound(t *testing.T) {
	expected := "XML Title"
	xmlSource := fmt.Sprintf(`<!DOCTYPE html>
	<html>
	<body>
	<h1>%s</h1>
	<h1>another node</h1>
	<p>My first paragraph.</p>
	</body>
	</html>`, expected)

	xe := htmlExtractor{}
	xpath := "//h2"
	_, err := xe.extractFromByteSlice([]byte(xmlSource), xpath)

	if err == nil {
		t.Errorf("TestHtmlExtraction_PathNotFound, should be err, got :%v", err)
	}
}

func TestInvalidHtml(t *testing.T) {
	xmlSource := `invalid html source`

	xe := htmlExtractor{}
	xpath := "//input"
	_, err := xe.extractFromByteSlice([]byte(xmlSource), xpath)

	if err == nil {
		t.Errorf("TestInvalidXml, should be err, got :%v", err)
	}
}

func TestHtmlComplexExtraction(t *testing.T) {
	expected := "Html Title"
	HtmlSource := fmt.Sprintf(`<!DOCTYPE html>
	<html>
	<body>
	<script>
		if (typeof resourceLoadedSuccessfully === "function") {
			resourceLoadedSuccessfully();
		}
		$(() => {
			typeof cssVars === "function" && cssVars({onlyLegacy: true});
		})
		var trackGeoLocation = false;
		alert('#@=$*€');
		</script>
	<h1>%s</h1>
	<p>My first paragraph.</p>
	</body>
	</html>`, expected)

	xe := htmlExtractor{}
	xpath := "//body/h1"
	val, err := xe.extractFromByteSlice([]byte(HtmlSource), xpath)

	if err != nil {
		t.Errorf("TestHtmlExtraction %v", err)
	}

	if !strings.EqualFold(val.(string), expected) {
		t.Errorf("TestHtmlExtraction expected: %s, got: %s", expected, val)
	}
}
