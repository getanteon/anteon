package tokenizer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func BenchmarkCount(b *testing.B) {
	dir := os.Getenv("DDOSIFY_TOKENIZERS_DIR")
	if dir == "" {
		b.Skip("NOTICE: DDOSIFY_TOKENIZERS_DIR is not set")
	}
	for _, name := range []string{"Qwen_Qwen3-0.6B", "unsloth_Llama-3.2-1B-Instruct", "TinyLlama_TinyLlama-1.1B-Chat-v1.0"} {
		start := time.Now()
		tok, err := LoadFile(filepath.Join(dir, name+".json"))
		if err != nil {
			b.Fatal(err)
		}
		b.Logf("%s loaded in %v", name, time.Since(start))
		text := strings.Repeat("Kredi kartı limitim ne kadar? The overdraft limit is 5,000 lira, and the fee is 2.5%. ", 50)
		b.Run(name, func(b *testing.B) {
			b.SetBytes(int64(len(text)))
			for i := 0; i < b.N; i++ {
				tok.Count(text)
			}
		})
	}
}
