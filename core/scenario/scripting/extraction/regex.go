package extraction

import (
	"fmt"
	"regexp"
)

type regexExtractor struct {
	r *regexp.Regexp
}

func (ri *regexExtractor) Init(regex string) {
	ri.r = regexp.MustCompile(regex)
}

func (ri *regexExtractor) extractFromString(text string, matchNo int) (string, error) {
	matches := ri.r.FindAllString(text, -1)

	if matches == nil {
		return "", fmt.Errorf("no match for the Regex: %s  Match no: %d", ri.r.String(), matchNo)
	}

	if len(matches) > matchNo {
		return matches[matchNo], nil
	}
	return matches[0], nil
}

func (ri *regexExtractor) extractFromByteSlice(text []byte, matchNo int) ([]byte, error) {
	matches := ri.r.FindAll(text, -1)

	if matches == nil {
		return nil, fmt.Errorf("no match for the Regex: %s  Match no: %d", ri.r.String(), matchNo)
	}

	if len(matches) > matchNo {
		return matches[matchNo], nil
	}
	return matches[0], nil
}
