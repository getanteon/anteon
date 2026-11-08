package config

import (
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"go.ddosify.com/ddosify/v2/core/scenario/scripting/injection"
	"go.ddosify.com/ddosify/v2/core/types/regex"
)

var envVarRe = regexp.MustCompile(regex.EnvironmentVariableRegex)

// ProxyURL resolves the {{$NAME}} OS environment variables in a proxy address with the injector
// the URL, headers and payload use, then parses it. An empty address means no proxy. Every
// unset variable is named at once.
func ProxyURL(addr string) (*url.URL, error) {
	if addr == "" {
		return nil, nil
	}
	var missing []string
	seen := map[string]bool{}
	for _, m := range envVarRe.FindAllString(addr, -1) {
		key := m[2 : len(m)-2]
		if !strings.HasPrefix(key, "$") || seen[key] {
			continue
		}
		seen[key] = true
		if _, ok := os.LookupEnv(key[1:]); !ok {
			missing = append(missing, key[1:])
		}
	}
	switch len(missing) {
	case 0:
	case 1:
		return nil, fmt.Errorf("the environment variable %s is not set", missing[0])
	default:
		return nil, fmt.Errorf("the environment variables %s are not set", strings.Join(missing, ", "))
	}

	ei := injection.EnvironmentInjector{}
	ei.Init()
	resolved, err := ei.InjectEnv(addr, map[string]interface{}{})
	if err != nil {
		return nil, err
	}
	return url.Parse(resolved)
}
