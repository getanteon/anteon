package injection

import (
	"encoding/json"
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
	injectStrFunc := func(s string) string {
		truncated := s[2 : len(s)-2] // {{...}}
		if env, ok := vars[truncated]; ok {
			return fmt.Sprint(env) // TODOcorr check string/interface{}
		}
		errors = append(errors, fmt.Errorf("%s could not be extracted from previous steps", truncated))
		return s // return back
	}

	// json injection
	if json.Valid([]byte(text)) {
		textJson := map[string]interface{}{}
		json.Unmarshal([]byte(text), &textJson)

		// keys
		for k, v := range textJson {
			if ri.r.MatchString(k) {
				replaced := ri.r.ReplaceAllStringFunc(k, injectStrFunc)
				textJson[replaced] = v
				delete(textJson, k)
			}
		}

		ri.replaceJson(textJson, vars)

		replacedBytes, err := json.Marshal(textJson)
		if err != nil || !json.Valid(replacedBytes) {
			return "", err
		}

		return string(replacedBytes), nil

	}
	// string injection
	replaced := ri.r.ReplaceAllStringFunc(text, injectStrFunc)
	if len(errors) == 0 {
		return replaced, nil
	}

	return replaced, unifyErrors(errors)

}

// recursive json replace
func (ri *RegexReplacer) replaceJson(textJson map[string]interface{}, vars map[string]interface{}) error {
	for k, v := range textJson { // check ints
		vv, isStr := v.(string)
		if isStr {
			if ri.r.MatchString(vv) {
				truncated := vv[2 : len(vv)-2]
				if env, ok := vars[truncated]; !ok {
					return fmt.Errorf("%s could not be extracted from previous steps", truncated)
				} else {
					if _, err := json.Marshal(env); err == nil {
						// object, set directly
						textJson[k] = env
						continue
					}

				}
			}
		} else if vv, isObject := v.(map[string]interface{}); isObject {
			ri.replaceJson(vv, vars)
		}

	}
	return nil
}

func unifyErrors(errors []error) error {
	sb := strings.Builder{}

	for _, err := range errors {
		sb.WriteString(err.Error())
	}

	return fmt.Errorf("%s", sb.String())
}
