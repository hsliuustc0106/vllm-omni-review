# vLLM-Omni Review Fork — Design

## Goal

Fork [alibaba/open-code-review](https://github.com/alibaba/open-code-review) into a domain-native review engine for `vllm-project/vllm-omni` PRs. Combine OCR's deterministic coverage (file-by-file, accurate line positioning, stable rule matching) with vllm-omni domain knowledge extracted from the existing [vllm-omni-review skill](https://github.com/vllm-project/vllm-omni).

## Motivation

The current skill-based approach uses a general-purpose agent with natural-language prompts. Problems:

- **Incomplete coverage** — agent skips files on larger changesets
- **Position drift** — reported issues don't match actual code locations
- **Unstable quality** — prompt variations cause inconsistent reviews
- **No domain automation** — every tensor shape, CUDA pattern, and connector lifecycle check relies on the LLM noticing it

OCR's deterministic engineering solves coverage and positioning. Our additions add domain intelligence through static analysis and targeted rule docs.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│  vllm-omni-review Skill (orchestration)              │
│  - Fetch PR, check gates, route to domain            │
│  - Run benchmarks on hardware                        │
│  - Post comments incrementally                       │
│  - Final verdict                                     │
└──────────────────┬──────────────────────────────────┘
                   │ calls ocr review
                   ▼
┌─────────────────────────────────────────────────────┐
│  OCR (this fork) — review engine                     │
│  1. Parse diffs + filter + categorize files          │
│  2. Static analysis → <analysis_hints>               │
│     - 10 regex patterns from blocker-patterns        │
│     - Tensor shape scanner                           │
│     - Platform pattern scanner                       │
│  3. [Plan phase] ← analysis hints injected           │
│  4. Main LLM loop ← vllm-omni rule docs applied      │
│  5. Collect + position comments                      │
└─────────────────────────────────────────────────────┘
```

## Changes

### 1. Static Analysis (`internal/analysis/` — new package)

Runs per file before the LLM sees the diff. Regex-based, ~50ms per file. Produces `<analysis_hints>` injected into the plan phase (or directly into main review for small diffs).

**10 regex checks (`analyzer.go`):**

| # | Check | Severity |
|---|-------|----------|
| 1 | Silent swallow (`except: pass`) | high |
| 2 | Bare except (`except:`) | medium |
| 3 | Blocking call in `async def` | high |
| 4 | Sequential await without `gather` | medium |
| 5 | Hardcoded secret/key/token | high |
| 6 | `eval()`/`exec()`/`pickle.loads()` | high |
| 7 | `open()` without `with` | medium |
| 8 | Mixin after `nn.Module` in MRO | medium |
| 9 | `.format()` on request/input variables | medium |
| 10 | Module-level mutable state in async | low |

**Tensor shape scanner (`shape.go`):**
- `.view()`/`.reshape()` — reports dimension changes
- Missing `.contiguous()` before `.view()`/`.permute()`/`.transpose()`
- dtype conversions (`.half()`/`.bfloat16()`/`.to(dtype=)`)
- `torch.cat()`/`torch.stack()` — notes tensors for shape verification

**Platform scanner (`platform.go`):**
- `torch.cuda.Stream()` without nearby `synchronize()`
- `.cuda()`/`.to('cuda')` — flags for memory lifecycle review
- `torch.empty()`/`torch.zeros()` with large allocation hints

### 2. Agent Wiring (`internal/agent/agent.go`)

In `executeSubtask()`, after rule resolution and before the plan phase:

```
resolve rule → run static analysis → [plan phase gets analysis hints] → main loop
```

For diffs under 50 lines (no plan phase), inject `<analysis_hints>` directly into the main review `<user_task>` block.

### 3. Domain Rule Docs (`internal/config/rules/rule_docs/`)

File categorization via glob patterns in `system_rules.json`:

| Layer | Glob Pattern | Rule Doc |
|-------|-------------|----------|
| Engine | `**/engine/**/*.py` | `engine.md` |
| Model Executor | `**/model_executor/**/*.py` | `models.md` |
| Diffusion | `**/diffusion/**/*.py` | `diffusion.md` |
| Entrypoints | `**/entrypoints/**/*.py` | `entrypoints.md` |
| Connectors | `**/connectors/**/*.py`, `**/distributed/omni_connectors/**` | `connectors.md` |
| Platforms | `**/platforms/**/*.py` | `platforms.md` |
| Quantization | `**/quantization/**/*.py` | `quantization.md` |
| Config | `**/config/**/*.{yaml,yml}` | `config.md` |
| General Python | `**/*.py` | `python_ml.md` (fallback) |

Each rule doc is a focused checklist (~20-30 lines) drawn from the vllm-omni-review skill's blocker-patterns and architecture references.

Key domain patterns covered: Mixin MRO, connector state management, async/sync path differences, stage config validation, diffusion latent cache lifecycle, input validation placement, tensor parallelism edge cases, test mock mismatches.

### 4. Task Template Rewrite (`internal/config/template/task_template.json`)

System prompt rewritten from Java/enterprise focus to vllm-omni maintainer:
- Knowledge of five-layer architecture and multi-stage pipelines
- Omni-modal data flow awareness (text, image, video, audio)
- Critical path focus: engine scheduler, model executor, connectors
- Multimodal correctness as a review dimension

Placeholder variables (`{{diff}}`, `{{change_files}}`, `{{system_rule}}`, `{{plan_guidance}}`) unchanged — no Go template code modified.

### 5. File Filters

Updated `default_exclude_patterns.json`:
```
**/tests/**, **/examples/**, **/benchmarks/**, **/third_party/**,
**/.venv/**, **/__pycache__/**, **/*.pyc, **/docker/**,
**/.buildkite/**, **/apps/**
```

`supported_file_types.json`: no changes needed (`.py` already supported). No `.cu`/`.cuh` files exist in vllm-omni.

### 6. Project-Level Config (`.opencodereview/rule.json`)

Optional override layer for repo-specific rules, placed at the fork root.

## What Doesn't Change

- Diff engine (`internal/diff/`)
- LLM client (`internal/llm/`) — Anthropic + OpenAI already supported
- Tools (`internal/tool/`) — existing 6 tools sufficient
- Session history (`internal/session/`)
- Viewer (`internal/viewer/`)
- CLI (`cmd/opencodereview/`) — command interface unchanged

## Estimated Effort

| Artifact | Approx. Size |
|----------|-------------|
| `internal/analysis/analyzer.go` | ~200 lines Go |
| `internal/analysis/shape.go` | ~150 lines Go |
| `internal/analysis/platform.go` | ~100 lines Go |
| `internal/agent/agent.go` changes | ~30 lines Go |
| 9 rule docs (`*.md`) | ~300 lines Markdown |
| `system_rules.json` update | ~20 lines JSON |
| `default_exclude_patterns.json` update | ~10 lines JSON |
| `task_template.json` rewrite | ~80 lines JSON |
| `.opencodereview/rule.json` | ~30 lines JSON |

Total: ~480 lines Go, ~300 lines Markdown, ~140 lines JSON.
