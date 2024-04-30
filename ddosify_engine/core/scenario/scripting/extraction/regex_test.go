package extraction

import (
	"strings"
	"testing"
)

func TestRegexExtractFromString(t *testing.T) {
	regex := "[a-z]+_[0-9]+"

	re := regexExtractor{}
	re.Init(regex)

	source := "messi_10alvarez_9"

	res, err2 := re.extractFromString(source, 1)
	if !strings.EqualFold(res, "alvarez_9") || err2 != nil {
		t.Errorf("RegexMatch should return second match")
	}

	res, err := re.extractFromString(source, 0)
	if !strings.EqualFold(res, "messi_10") || err != nil {
		t.Errorf("RegexMatch should return first match")
	}

}

func TestRegexExtractFromStringNoMatch(t *testing.T) {
	regex := "[a-z]+_[0-9]+"

	re := regexExtractor{}
	re.Init(regex)

	source := "messialvarez"

	_, err := re.extractFromString(source, 0)
	if err == nil {
		t.Errorf("Should be error %v", err)
	}

}
