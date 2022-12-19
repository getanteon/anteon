package scripting

import (
	"fmt"
	"regexp"
	"strings"
)

type RegexReplacer struct {
	r *regexp.Regexp
}

func CreateRegexReplacer(regex string) *RegexReplacer {
	return &RegexReplacer{
		r: regexp.MustCompile(regex),
	}
}

func (ri *RegexReplacer) Inject(text string, vars map[string]interface{}) (string, error) {
	errors := []error{}
	injectFunc := func(s string) string {
		truncated := s[2 : len(s)-2] // {{...}}
		if env, ok := vars[truncated]; ok {
			return env.(string) // TODOcorr check string/interface{}
		} else {
			errors = append(errors, fmt.Errorf("%s could not be extracted from previous steps", truncated))
			return s // return back
		}
	}

	replaced := ri.r.ReplaceAllStringFunc(text, injectFunc)
	if len(errors) == 0 {
		return replaced, nil
	} else {
		return replaced, unifyErrors(errors)
	}
}

func unifyErrors(errors []error) error {
	sb := strings.Builder{}

	for _, err := range errors {
		sb.WriteString(err.Error())
	}

	return fmt.Errorf("%s", sb.String())
}
