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

func (ri *RegexExtractor) ExtractFromString(text string, matchNo int) (interface{}, error) {
	matches := ri.r.FindAllString(text, -1)

	if matches == nil {
		return nil, fmt.Errorf("No match")
	}

	if len(matches) > matchNo {
		return matches[matchNo], nil
	}
	return matches[0], nil
}
