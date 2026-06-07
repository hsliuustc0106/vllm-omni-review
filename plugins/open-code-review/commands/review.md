---
description: Run vomni-review to review code changes and autonomously apply fixes.
---

Invoke the code review engine vomni-review to review current code changes, and let the Agent autonomously decide whether to apply fixes.

## Workflow

### Step 1: Run Code Review

Run the vomni-review command:

```bash
vomni-review review --audience agent [user-args]
```
- Default (no user arguments): reviews staged, unstaged, and untracked changes (workspace mode).
- If the user provides `--commit` or `--c`: pass through as-is.
- If the user provides `--from` and `--to`: pass through as-is.
- (Optional) Provide `--background "requirement context"` to review whether the requirements are correctly implemented.
- Capture full stdout. Set a 5-minute timeout.
- If the `vomni-review` command is not found, build it from source with `make build`.

### Step 2: Filter and Evaluate

For each comment, assess its validity and quality:

- **High**: Obvious bugs, security issues, clear mistakes, or well-founded suggestions with precise fix proposals
- **Medium**: Reasonable concerns but context-dependent, style/performance suggestions, or fixes that require manual implementation
- **Low**: Likely false positives, lacking sufficient context, nitpicks, or meaningless suggestions

Silently discard low-confidence comments. Display the remaining comments.

### Step 3: Fix

Automatically fix issues and suggestions that are worth adopting.