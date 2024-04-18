package regex

import (
	"regexp"
	"testing"
)

func TestDynamicVariableRegex(t *testing.T) {
	re := regexp.MustCompile(DynamicVariableRegex)
	// Sub Tests
	tests := []struct {
		name        string
		url         string
		shouldMatch bool
	}{
		{"Match1", "https://example.com/{{_abc}}", true},
		{"Match2", "https://example.com/{{_timestamp}}", true},
		{"Match3", "https://example.com/aaa/{{_timestamp}}", true},
		{"Match4", "https://example.com/aaa/{{_timestamp}}/bbb", true},
		{"Match5", "https://example.com/{{_timestamp}}/{_abc}", true},
		{"Match6", "https://example.com/{{_abc/{{_timestamp}}", true},
		{"Match7", "https://example.com/_aaa/{{_timestamp}}", true},

		{"Not Match1", "https://example.com/{{_abc", false},
		{"Not Match2", "https://example.com/{{_abc}", false},
		{"Not Match3", "https://example.com/_abc", false},
		{"Not Match4", "https://example.com/{{abc", false},
		{"Not Match5", "https://example.com/abc", false},
		{"Not Match6", "https://example.com/abc/{{cc}}", false},
		{"Not Match7", "https://example.com/abc/{{cc}}/fcf", false},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			matched := re.MatchString(test.url)

			if test.shouldMatch != matched {
				t.Errorf("Name: %s, ShouldMatch: %t, Matched: %t\n", test.name, test.shouldMatch, matched)
			}

		}
		t.Run(test.name, tf)
	}
}

func TestEnvironmentVariableRegex(t *testing.T) {
	re := regexp.MustCompile(EnvironmentVariableRegex)
	// Sub Tests
	tests := []struct {
		name        string
		url         string
		shouldMatch bool
	}{
		{"Match1", "{{a}}", true},
		{"Match2", "{{ab}}", true},
		{"Match3", "{{ab1}}", true},
		{"Match4", "{{Ab1}}/bbb", true},
		{"Match5", "{{ABC}}/{_abc}", true},
		{"Match6", "{{_abc/{{ABC__fc_111}}", true},
		{"Match7", "{{a_b}}", true},
		{"Match8", "xx{{a}}", true},
		{"Match9", "{{a}}bb", true},
		{"Match10", "cx{{a}}vc", true},
		{"Match10", "cx {{a}}vc", true},
		{"Match11", "cx{{a}} vc", true},
		{"Match11", "cx{{a}}_-", true},
		{"Match12", "{{a-v}}", true},
		{"Match13", "{{AV-}}", true},

		{"Not Match1", "{{}}", false},
		{"Not Match2", "{{_abc}}", false},
		{"Not Match4", "{{abc!}}", false},
		{"Not Match6", "{{_A}}", false},
		{"Not Match7", "{{_AB_2}}", false},
		{"Not Match8", "{{£AB_2}}", false},
		{"Not Match8", "{{3AB_2}}", false},
		{"Not Match8", "{{%3AB_2}}", false},
	}

	for _, test := range tests {
		tf := func(t *testing.T) {
			matched := re.MatchString(test.url)

			if test.shouldMatch != matched {
				t.Errorf("Name: %s, ShouldMatch: %t, Matched: %t\n", test.name, test.shouldMatch, matched)
			}

		}
		t.Run(test.name, tf)
	}
}
