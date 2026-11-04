# LLM Load Testing

Ddosify measures an LLM endpoint the way the serving benchmarks do ([vLLM `bench serve`](https://docs.vllm.ai/en/latest/benchmarking/cli/), [NVIDIA AIPerf](https://docs.nvidia.com/nim/benchmarking/llm/latest/metrics.html), [GuideLLM](https://github.com/vllm-project/guidellm)), with latency and token percentiles over the successful requests only, as they aggregate them, while keeping what it already does well: an open arrival rate, multi-step scenarios, captures, test data and stages. Two ways in:

- **The OpenAI preset**: a URL, a key and a dataset, for any OpenAI compatible chat completions server (vLLM, SGLang, LiteLLM, TGI, llama.cpp, OpenAI).
- **Any SSE API**: an `llm` block on a streamed step, for an application API such as an AI SDK chat endpoint.

Both build on [the stream config and its metrics](../README.md#streaming-sse-and-llm-endpoints): TTFB, first event, first token, total, gaps, ends, goodput and stages.

## Counting output tokens

Every streamed answer is counted from the first source that is available:

1. **The server's usage.** An OpenAI compatible stream carries it in its last chunk when the request asks for `"stream_options": {"include_usage": true}`, which the preset does. Also read: the Responses API's `response.completed`, Anthropic's `message_start` and `message_delta`, and the AI SDK UI message stream's `messageMetadata.usage` or `messageMetadata.totalUsage` (`outputTokens`, `inputTokens`).
2. **A tokenizer.** `"tokenizer": "path/to/tokenizer.json"`, the Hugging Face file of the served model, counts the streamed text. Pure Go, no cgo: byte-level BPE (Llama 3, Qwen 2 and 3, GPT-2 style) and SentencePiece style BPE with byte fallback (Llama 2, Mistral, TinyLlama). Against Hugging Face `tokenizers` 0.23.3 it gives the same ids for all 320 test strings (English, Turkish, code, digits, whitespace runs, emoji, CJK, special tokens) on Qwen3, Qwen3.8 (27B FP8), Qwen2.5, Llama 3.2 and TinyLlama. A WordPiece or Unigram model, or a split pattern with a lookaround other than `\s+(?!\S)`, is refused at start.
3. **Content events.** With neither, the count is the number of events that carried text, and the report says **chunks**, never tokens: a chunk can hold several tokens.

`"token_source"` forces one: `usage`, `tokenizer` or `chunks` (`auto` is the order above). The text is read from the OpenAI `choices[0].delta.content` (and `reasoning` or `reasoning_content`), `choices[0].text`, the AI SDK `text-delta` and `reasoning-delta`, Anthropic `content_block_delta` and the Responses API `response.output_text.delta`; `"text_paths"` and `"usage": {"output": "...", "input": "..."}` take [gjson](https://github.com/tidwall/gjson) paths for another format.

**Gurubase**: its `finish` usage is what the whole run spent, every model call of planning, writing and judging, not the tokens of the streamed answer. Use `"token_source": "tokenizer"` there, so TPOT is the answer's; the server's numbers stay in each result as `usage_output` and `usage_input`.

## Metrics

| Metric | Definition |
| --- | --- |
| First content | Time to the first event that carried output text, reasoning included. AIPerf's TTFT ignores empty first responses the same way |
| TPOT | (last content event - first content event) / (output tokens - 1), per request, then its percentiles. As vLLM and AIPerf define it; the trailing usage chunk and `[DONE]` are left out |
| ITL | The time between consecutive content events, every successful request pooled, as `vllm bench serve` reports it. With one token per chunk it is the inter-token latency; tokens sharing a chunk add no gap |
| Tokens/s/request | Output tokens / time to the last content event, AIPerf's per user TPS |
| Output throughput | All output tokens / (last answer's end - first request's send), warm-up excluded, AIPerf's system TPS |
| Input tokens | The server's usage, or the tokenizer on the prompt text (the chat template is not added, so it is a little low) |

The text report prints them per step (and per stage in the JSON output) after the stream table; `-o stdout-json` puts them under `steps.<id>.stream.llm`, `stages[].stream.llm` and, for all steps together, `llm`. `"slo": {"tpot_ms": 50}` makes TPOT a goodput threshold; a request counted in chunks has no TPOT in tokens and misses it.

## The OpenAI preset

```json
{
    "load_spread": "even",
    "manual_load": [
        {"duration": 60, "count": 30, "name": "warm-up", "warmup": true},
        {"duration": 300, "count": 300, "name": "1/s"},
        {"duration": 300, "count": 600, "name": "2/s"}
    ],
    "slo": {"first_token_ms": 2000, "tpot_ms": 50},
    "llm": {
        "preset": "openai",
        "url": "http://localhost:8000/v1/chat/completions",
        "api_key": "{{$OPENAI_API_KEY}}",
        "model": "Qwen/Qwen3-8B",
        "max_tokens": 256,
        "dataset": {"path": "prompts.jsonl"},
        "tokenizer": "tokenizer.json",
        "cache_bust": true
    }
}
```

The top level `llm` replaces `steps`. Each request is `{"model", "messages", "stream": true, "stream_options": {"include_usage": true}, "max_tokens"}` with `Authorization: Bearer <api_key>`.

| Field | Meaning | Default |
| --- | --- | --- |
| `preset` | `openai` | `openai` |
| `url` | The chat completions URL | required |
| `api_key` | Sent as a bearer token; `{{$NAME}}` reads the environment | none |
| `headers` | More headers | none |
| `model`, `max_tokens`, `temperature`, `system_prompt` | Into the body | none |
| `max_tokens_field` | `max_completion_tokens` for an OpenAI reasoning model | `max_tokens` |
| `ignore_eos` | vLLM's `ignore_eos`, so every answer runs to `max_tokens` | `false` |
| `include_usage` | Ask for the usage chunk | `true` |
| `extra_body` | Merged into the body last | none |
| `dataset` | `{"path", "format", "field", "fields", "header", "limit"}` | |
| `prompts` | Inline prompts, besides or instead of `dataset` | |
| `order` | `sequential` or `random` | `sequential` |
| `turns`, `think_time` | A session of this many turns, the pause between them in ms (`"2000"` or `"1000-3000"`) | `1` |
| `timeout` | Seconds for the whole stream | `300` |
| `tokenizer`, `token_source`, `cache_bust`, `shared_prefix_tokens` | As below | |

A dataset is JSONL (`{"prompt": ...}`, `{"text": ...}`, `{"turns": [...]}`, OpenAI `{"messages": [...]}` whose user messages are the turns, ShareGPT `{"conversations": [...]}`, or `"field"`), CSV (`"field"` a column name or index, `"fields"` the columns of a session's turns) or plain text, one prompt a line. The format follows the extension unless `format` says otherwise.

## Any SSE API

```json
{
    "id": 2,
    "url": "{{$GURUBASE_URL}}/api/v1beta/gurus/{{$GURU_ID}}/chats/{{CHAT_ID}}/messages",
    "method": "POST",
    "payload": "{\"message\": {\"role\": \"user\", \"parts\": [{\"type\": \"text\", \"text\": \"{{data.sessions.question}}\"}]}}",
    "stream": {"type": "sse"},
    "llm": {
        "tokenizer": "tokenizer.json",
        "token_source": "tokenizer",
        "prompt_path": "message.parts.0.text",
        "capture_text": "ANSWER",
        "capture": {"RUN_ID": "messageMetadata.runId"}
    }
}
```

| Field | Meaning |
| --- | --- |
| `tokenizer`, `token_source`, `text_paths`, `usage` | Counting, as above |
| `prompt_path` | gjson path to the prompt string in the body, after variables are injected. Needed by `cache_bust` and `shared_prefix_tokens`; also counted as input tokens |
| `cache_bust` | A unique marker at the start of the prompt |
| `shared_prefix_tokens` | A fixed prefix of this many tokens before the prompt |
| `capture_text` | A variable that gets the answer's text |
| `capture` | Variables from the first event holding a gjson path, such as a run id in the AI SDK `start` event |

A step with `llm` and no `stream` gets the default stream config. [config_examples/llm_sse_session.json](../config_examples/llm_sse_session.json) is a two turn Gurubase session.

## Multi-turn sessions

An iteration is a session: its steps run one after another, a step's `sleep` is the think time before the next, and captured variables carry into the next request.

- **The preset** with `"turns": N` makes N steps. Each turn sends the earlier turns' user and assistant messages, then its own: turn k of a dataset row with turns is that turn; a row with fewer turns borrows the first turn of the rows after it.
- **An API that keeps the history**, such as Gurubase, carries an id: capture the chat id from the first response (`capture_env`) or from the stream (`llm.capture`), and use it in the next URL.
- **An API that wants the history in the body** gets the answer from `capture_text`, put into the next payload as `"{{ANSWER}}"`, quoted, so it is escaped as a JSON string.

## Prefix caching: why repeated prompts inflate results

vLLM's automatic prefix caching and SGLang's radix cache keep the KV of prompts they have seen and skip the prefill of the longest prefix a new prompt shares with one of them. A load test that asks the same 50 questions a thousand times measures a cache: after the first round every prompt is a hit, the time to first token is the time to prefill the few tokens after the shared part, and the card seems to hold several times the users it would hold for real traffic. vLLM's documentation warns that rerunning a benchmark against the same server inflates throughput this way.

- **`cache_bust`** puts `[rid:<16 hex>]` and a blank line at the start of the prompt, as AIPerf's `first_turn_prefix` target does. The 16 hex digits are a random run nonce and a counter, so no two requests of a run share it and a new run does not hit the cache the last one left. In the preset the marker goes on a session's first user turn, which later turns resend, so a session still reuses its own history, as a real conversation would; a system prompt stays shared. With `prompt_path` every request gets a new one.
- **`shared_prefix_tokens`** puts the same text of N tokens before every prompt, as vLLM's `prefix_repetition` dataset does, to measure a known cacheable share. With a tokenizer it is N tokens as that tokenizer counts them; without one it is N common words, about N tokens on a Llama 3 or Qwen vocabulary. In the preset it is the system message; with `prompt_path` it comes before the marker, so the prefix stays cacheable and the rest does not.

What fits depends on the question. Sizing a card for real users: `cache_bust`, plus a shared prefix the size of the real system prompt. Measuring the cache itself: two runs, with and without. An application that puts its own system prompt and documents first (Gurubase) already varies the prompt; a marker in the user's question changes the question the application reads, so leave `cache_bust` off there unless the answers are not being judged.

## Coordinated omission: scheduling lag

Ddosify's arrival rate is an open model: an iteration starts on schedule whether or not the earlier ones have answered. When the client itself falls behind (CPU, garbage collection, waiting for a pooled client) the first request goes out late. The server then sees a lower rate than asked, and every latency leaves out the wait: coordinated omission.

Each iteration records when the schedule meant it to start and when its first request was sent. The report prints the lag's p50, p99 and max and the iterations more than 100 ms late (`schedule` in the JSON), and warns when the p99 is above 100 ms: the numbers of that run describe a slower test than the one configured. Run from a bigger machine, or split the load.

## From the command line

The preset needs no config file:

```bash
ddosify run --preset openai --url http://localhost:8000/v1/chat/completions --model Qwen/Qwen3-8B \
  --dataset config_examples/prompts.jsonl --tokenizer tokenizer.json --max-tokens 256 -n 120 -d 60
```

`--url`, `--model`, `--dataset`, `--api-key` and `--max-tokens` fill the top level `llm` object, over a config's when `--config` is given too; `DDOSIFY_API_KEY` keeps the key off the command line. `--tokenizer` (`DDOSIFY_TOKENIZER`) gives every streamed step that tokenizer, and gives a streamed step without an `llm` block one, so a plain SSE scenario counts tokens too.

## Gurubase

[config_examples/sse_llm.json](../config_examples/sse_llm.json) is a Gurubase test: create a chat, ask a question from a CSV, a warm-up and stepped stages, an SLO. Its ask step says `"token_source": "tokenizer"` because the `finish` chunk's usage is the whole run's spend, not the answer's (see [Counting output tokens](#counting-output-tokens)). Give the served model's `tokenizer.json` with `--tokenizer`.

## Where the metrics show

The live TUI and NDJSON lines (`ddosify run` on a terminal, or in a pipe) show TPOT, output per second, ITL, the prefix cache setting and the scheduling lag next to TTFT; `--html report.html` writes them into the report's "Answers' output" section. Output counted in content events is labelled chunks everywhere, and TPOT is then not shown.
