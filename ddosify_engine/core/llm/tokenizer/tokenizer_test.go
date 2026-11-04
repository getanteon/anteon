package tokenizer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

// testdata/reference.json holds the counts and ids Hugging Face tokenizers 0.23.3 gave for each string,
// with encode(s, add_special_tokens=False). The tokenizer.json files are too large to commit: set
// DDOSIFY_TOKENIZERS_DIR to a directory holding <name>.json, the name being the model id with "/" as "_",
// downloaded from https://huggingface.co/<model id>/resolve/main/tokenizer.json.
type reference struct {
	Strings    []string `json:"strings"`
	Tokenizers map[string][]struct {
		Count int   `json:"count"`
		IDs   []int `json:"ids"`
	} `json:"tokenizers"`
}

func TestReferenceCounts(t *testing.T) {
	dir := os.Getenv("DDOSIFY_TOKENIZERS_DIR")
	if dir == "" {
		t.Skip("NOTICE: DDOSIFY_TOKENIZERS_DIR is not set, the Hugging Face reference counts are not checked")
	}
	data, err := os.ReadFile("testdata/reference.json")
	if err != nil {
		t.Fatal(err)
	}
	var ref reference
	if err := json.Unmarshal(data, &ref); err != nil {
		t.Fatal(err)
	}
	for name, cases := range ref.Tokenizers {
		name, cases := name, cases
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name+".json")
			if _, err := os.Stat(path); err != nil {
				t.Skipf("NOTICE: %s not found", path)
			}
			tok, err := LoadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			for i, s := range ref.Strings {
				got := tok.Encode(s)
				if len(got) != cases[i].Count {
					t.Errorf("string %d %.40q: %d tokens, want %d (got %v, want %v)", i, s, len(got), cases[i].Count, got, cases[i].IDs)
					continue
				}
				if cases[i].IDs != nil && !reflect.DeepEqual(got, cases[i].IDs) && len(got) > 0 {
					t.Errorf("string %d %.40q: ids %v, want %v", i, s, got, cases[i].IDs)
				}
			}
		})
	}
}

// A byte-level BPE small enough to follow by hand.
const tinyByteLevel = `{
 "added_tokens": [{"id": 100, "content": "<|end|>"}],
 "normalizer": {"type": "NFC"},
 "pre_tokenizer": {"type": "Sequence", "pretokenizers": [
  {"type": "Split", "pattern": {"Regex": "(?i:'s|'t|'re|'ve|'m|'ll|'d)|[^\\r\\n\\p{L}\\p{N}]?\\p{L}+|\\p{N}| ?[^\\s\\p{L}\\p{N}]+[\\r\\n]*|\\s*[\\r\\n]+|\\s+(?!\\S)|\\s+"}, "behavior": "Isolated", "invert": false},
  {"type": "ByteLevel", "add_prefix_space": false, "use_regex": false}]},
 "model": {"type": "BPE", "byte_fallback": false, "ignore_merges": false,
  "vocab": {"h": 0, "e": 1, "l": 2, "o": 3, "Ġ": 4, "w": 5, "r": 6, "d": 7, "he": 8, "ll": 9, "hell": 10, "hello": 11, "Ġw": 12, "or": 13, "Ġwor": 14, "Ġworld": 15, "ĠĠ": 16, "1": 17, "2": 18, "!": 19, "Ġwo": 20, "Ġworl": 21},
  "merges": ["h e", "l l", "he ll", "hell o", ["Ġ", "w"], ["Ġw", "o"], ["Ġwo", "r"], ["Ġwor", "l"], ["Ġworl", "d"], "Ġ Ġ"]}
}`

func TestTinyByteLevel(t *testing.T) {
	tok, err := Parse([]byte(tinyByteLevel))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		in   string
		want []int
	}{
		{"hello world", []int{11, 15}},
		{"hello   world", []int{11, 16, 15}}, // "  " then " world": the lookahead keeps one space for the word
		{"12!", []int{17, 18, 19}},
		{"hello<|end|>world", []int{11, 100, 5, 3, 6, 2, 7}},
		{"", []int{}},
	}
	for _, c := range cases {
		got := tok.Encode(c.in)
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%q: got %v, want %v", c.in, got, c.want)
		}
	}
}

func TestConcurrentEncode(t *testing.T) {
	tok, err := Parse([]byte(tinyByteLevel))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				if n := tok.Count("hello world hello   world"); n != 6 {
					t.Errorf("count %d, want 6", n)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestUnsupportedPattern(t *testing.T) {
	bad := `{"pre_tokenizer": {"type": "Split", "pattern": {"Regex": "a(?=b)"}, "behavior": "Isolated"}, "model": {"type": "BPE", "vocab": {"a": 0}, "merges": []}}`
	if _, err := Parse([]byte(bad)); err == nil {
		t.Fatal("a lookahead other than \\s+(?!\\S) must be refused")
	}
	wp := `{"model": {"type": "WordPiece", "vocab": {"a": 0}}}`
	if _, err := Parse([]byte(wp)); err == nil {
		t.Fatal("WordPiece must be refused")
	}
}
