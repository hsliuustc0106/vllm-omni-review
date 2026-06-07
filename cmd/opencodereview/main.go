// vomni-review is a domain-native code review engine for vllm-omni PRs.
// It reads git diffs, runs static analysis, and uses an LLM agent to generate structured review comments.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/open-code-review/open-code-review/internal/llm"
	"github.com/open-code-review/open-code-review/internal/telemetry"
)

func main() {
	llm.AppVersion = Version
	llm.InitEmbeddedLoader()

	ctx := context.Background()
	if telemetry.Init(ctx) {
		defer telemetry.ShutdownWithTimeout(ctx, 5*time.Second)
	}

	if err := dispatch(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// dispatch routes top-level subcommands or global flags.
func dispatch() error {
	args := os.Args[1:]

	// No args → default to review with empty args (will trigger usage/help)
	if len(args) == 0 {
		printTopLevelUsage()
		return nil
	}

	switch args[0] {
	case "--version", "-V":
		printVersion()
		return nil
	case "version":
		printVersion()
		return nil
	case "review", "r":
		return runReview(args[1:])
	case "config":
		return runConfig(args[1:])
	case "llm":
		return runLLM(args[1:])
	case "rules":
		return runRules(args[1:])
	case "viewer":
		return runViewer(args[1:])
	case "-h", "--help":
		printTopLevelUsage()
		return nil
	default:
		return fmt.Errorf("unknown command: %s\nRun 'ocr' for usage", args[0])
	}
}

func printTopLevelUsage() {
	fmt.Println(`vomni-review - Domain-native code review engine for vllm-omni

Usage:
  vomni-review [command]

Commands:
  review, r    Start a code review
  rules        Inspect and debug review rules
  config       Manage configuration settings
  llm          LLM utility commands
  viewer       Start the WebUI session viewer
  version      Show version information

Examples:
  vomni-review review --from main --to feature    Review diff range
  vomni-review review --commit abc123             Review a single commit
  vomni-review config set llm.model claude-opus-4-6  Set a config value
  vomni-review llm test                           Test LLM connectivity
  vomni-review version                            Show version info

Use "vomni-review review -h" for more information about review.
Use "vomni-review rules -h" for more information about rules.
Use "vomni-review config" for more information about config.
Use "vomni-review llm" for more information about LLM utilities.`)
}
