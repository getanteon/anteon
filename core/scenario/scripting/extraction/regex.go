package extraction

import (
	"fmt"
	"regexp"
)

type RegexExtractor struct {
	r *regexp.Regexp
}

func CreateRegexExtractor(regex string) *RegexExtractor {
	return &RegexExtractor{
		r: regexp.MustCompile(regex),
	}
}

func (ri *RegexExtractor) extractFromString(text string, matchNo int) (string, error) {
	matches := ri.r.FindAllString(text, -1)

	if matches == nil {
		return "", fmt.Errorf("No match")
	}

	if len(matches) > matchNo {
		return matches[matchNo], nil
	}
	return matches[0], nil
}

func (ri *RegexExtractor) extractFromByteSlice(text []byte, matchNo int) ([]byte, error) {
	matches := ri.r.FindAll(text, -1)

	if matches == nil {
		return nil, fmt.Errorf("No match")
	}

	if len(matches) > matchNo {
		return matches[matchNo], nil
	}
	return matches[0], nil
}
