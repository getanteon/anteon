package injection

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"go.ddosify.com/ddosify/core/types/regex"
)

type EnvironmentInjector struct {
	r  *regexp.Regexp
	jr *regexp.Regexp
}

func (ei *EnvironmentInjector) Init() {
	ei.r = regexp.MustCompile(regex.EnvironmentVariableRegex)
	ei.jr = regexp.MustCompile(regex.JsonEnvironmentVarRegex)
}

func (ei *EnvironmentInjector) Inject(text string, vars map[string]interface{}) (string, error) {
	errors := []error{}
	injectStrFunc := func(s string) string {
		truncated := s[2 : len(s)-2] // {{...}}
		if env, ok := vars[truncated]; ok {
			switch env.(type) {
			case string:
				return env.(string)
			case []byte:
				return string(env.([]byte))
			case int64:
				return fmt.Sprintf("%d", env)
			case int:
				return fmt.Sprintf("%d", env)
			case float64:
				return fmt.Sprintf("%f", env)
			case bool:
				return fmt.Sprintf("%t", env)
			default:
				return fmt.Sprint(env)
			}
		}
		errors = append(errors,
			fmt.Errorf("%s could not be found in vars global and extracted from previous steps", truncated))
		return s
	}
	injectToJsonByteFunc := func(s []byte) []byte {
		truncated := s[3 : len(s)-3] // "{{...}}"
		if env, ok := vars[string(truncated)]; ok {
			mEnv, _ := json.Marshal(env)
			return mEnv
		}
		errors = append(errors,
			fmt.Errorf("%s could not be found in vars global and extracted from previous steps", truncated))
		return s
	}

	// json injection
	bText := []byte(text)
	if json.Valid(bText) {
		if ei.jr.Match(bText) {
			replacedBytes := ei.jr.ReplaceAllFunc(bText, injectToJsonByteFunc)
			return string(replacedBytes), nil
		}
	}

	// string injection
	replaced := ei.r.ReplaceAllStringFunc(text, injectStrFunc)
	if len(errors) == 0 {
		return replaced, nil
	}

	return replaced, unifyErrors(errors)

}

func unifyErrors(errors []error) error {
	sb := strings.Builder{}

	for _, err := range errors {
		sb.WriteString(err.Error())
	}

	return fmt.Errorf("%s", sb.String())
}
