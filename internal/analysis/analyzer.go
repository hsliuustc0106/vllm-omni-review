// Package analysis provides static analysis of code diffs to produce
// attention hints for the LLM review phase. All checks are regex-based
// and run in under 50ms per file.
package analysis

import (
	"regexp"
	"strings"
)

// Hint describes a single finding from static analysis.
type Hint struct {
	Severity string // high, medium, low
	Pattern  string // e.g., "silent-swallow", "blocking-in-async"
	Line     int    // approximate line number in the diff
	Snippet  string // relevant code snippet
	Message  string // human-readable description
}

// AnalyzePython runs all Python-specific checks on a diff and returns hints.
func AnalyzePython(diff string) []Hint {
	var hints []Hint
	lines := strings.Split(diff, "\n")

	hints = append(hints, checkSilentSwallow(lines)...)
	hints = append(hints, checkBareExcept(lines)...)
	hints = append(hints, checkBlockingInAsync(lines)...)
	hints = append(hints, checkSequentialAwait(lines)...)
	hints = append(hints, checkHardcodedSecret(lines)...)
	hints = append(hints, checkEvalExec(lines)...)
	hints = append(hints, checkUnclosedResource(lines)...)
	hints = append(hints, checkMixinMRO(lines)...)
	hints = append(hints, checkFormatOnInput(lines)...)
	hints = append(hints, checkModuleMutableState(lines)...)

	return hints
}

var silentSwallowRE = regexp.MustCompile(`except\s*:?\s*$`)

func checkSilentSwallow(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		if silentSwallowRE.MatchString(strings.TrimSpace(line)) {
			for j := i + 1; j < len(lines) && j < i+3; j++ {
				if stripDiff(lines[j]) == "pass" {
					hints = append(hints, Hint{
						Severity: "high",
						Pattern:  "silent-swallow",
						Line:     i + 1,
						Snippet:  strings.TrimSpace(line),
						Message:  "Silent exception swallow: bare except followed by pass",
					})
					break
				}
				if stripDiff(lines[j]) != "" && !strings.HasPrefix(stripDiff(lines[j]), "#") {
					break
				}
			}
		}
	}
	return hints
}

var bareExceptRE = regexp.MustCompile(`except\s*:`)

func checkBareExcept(lines []string) []Hint {
	var hints []Hint
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if bareExceptRE.MatchString(trimmed) && !strings.HasPrefix(trimmed, "except:") {
			continue
		}
		// Only report bare 'except:' (already handled by silent-swallow if followed by pass)
	}
	return hints
}

var blockingCalls = []string{
	"time.sleep(", "requests.get(", "requests.post(", "requests.put(",
	"requests.delete(", "requests.patch(", "urllib.",
	"subprocess.run(", "subprocess.call(", "subprocess.Popen(",
	"os.system(",
}

func checkBlockingInAsync(lines []string) []Hint {
	var hints []Hint
	inAsync := false
	for i, line := range lines {
		code := stripDiff(line)
		if strings.HasPrefix(code, "async def ") {
			inAsync = true
			continue
		}
		if inAsync && strings.HasPrefix(code, "def ") && !strings.HasPrefix(code, "def __") {
			inAsync = false
			continue
		}
		if inAsync {
			for _, bc := range blockingCalls {
				if strings.Contains(code, bc) {
					hints = append(hints, Hint{
						Severity: "high",
						Pattern:  "blocking-in-async",
						Line:     i + 1,
						Snippet:  code,
						Message:  "Blocking call in async function: use asyncio.to_thread() or await equivalent",
					})
					break
				}
			}
		}
	}
	return hints
}

func checkSequentialAwait(lines []string) []Hint {
	var hints []Hint
	inAsync := false
	inLoop := false
	hasAwait := false
	var loopStart int

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "async def ") {
			inAsync = true
			inLoop = false
			hasAwait = false
			continue
		}
		if inAsync && strings.HasPrefix(trimmed, "def ") {
			inAsync = false
			continue
		}
		if !inAsync {
			continue
		}
		if strings.HasPrefix(trimmed, "for ") && strings.Contains(trimmed, " in ") {
			inLoop = true
			hasAwait = false
			loopStart = i + 1
			continue
		}
		if inLoop && strings.Contains(trimmed, "await ") {
			hasAwait = true
		}
		if inLoop && (trimmed == "" || (!strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") && trimmed != "")) {
			if hasAwait && !strings.Contains(strings.Join(lines[loopStart-1:i+1], "\n"), "asyncio.gather") {
				hints = append(hints, Hint{
					Severity: "medium",
					Pattern:  "sequential-await",
					Line:     loopStart,
					Snippet:  strings.TrimSpace(lines[loopStart-1]),
					Message:  "Sequential await in loop: consider asyncio.gather() for parallelism",
				})
			}
			inLoop = false
			hasAwait = false
		}
	}
	return hints
}

var secretPatterns = regexp.MustCompile(`(?i)(api_key|apikey|password|secret|token|auth)\s*[:=]\s*["'][^"'\s]{16,}["']`)

func checkHardcodedSecret(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		if secretPatterns.MatchString(line) {
			hints = append(hints, Hint{
				Severity: "high",
				Pattern:  "hardcoded-secret",
				Line:     i + 1,
				Snippet:  strings.TrimSpace(line),
				Message:  "Hardcoded secret detected: use environment variables instead",
			})
		}
	}
	return hints
}

var dangerousCalls = []string{"eval(", "exec(", "pickle.loads("}

func checkEvalExec(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, dc := range dangerousCalls {
			if strings.Contains(trimmed, dc) {
				hints = append(hints, Hint{
					Severity: "high",
					Pattern:  "dangerous-call",
					Line:     i + 1,
					Snippet:  trimmed,
					Message:  dc + " detected: potential code injection risk",
				})
			}
		}
	}
	return hints
}

func checkUnclosedResource(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "open(") && !strings.Contains(trimmed, "with ") {
			inWith := false
			for j := i - 1; j >= 0 && j >= i-3; j-- {
				if strings.Contains(strings.TrimSpace(lines[j]), "with ") && strings.HasSuffix(strings.TrimSpace(lines[j]), ":") {
					inWith = true
					break
				}
			}
			if !inWith {
				hints = append(hints, Hint{
					Severity: "medium",
					Pattern:  "unclosed-resource",
					Line:     i + 1,
					Snippet:  trimmed,
					Message:  "open() without 'with' statement: resource may not be closed",
				})
			}
		}
	}
	return hints
}

var mixinMROPattern = regexp.MustCompile(`class\s+\w+\([^)]*nn\.Module[^)]*,\s*\w*Mixin`)

func checkMixinMRO(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		if mixinMROPattern.MatchString(line) {
			hints = append(hints, Hint{
				Severity: "medium",
				Pattern:  "mixin-mro",
				Line:     i + 1,
				Snippet:  strings.TrimSpace(line),
				Message:  "Mixin after nn.Module in MRO: mixin __init__ may not be called. Move mixin before nn.Module or use lazy init.",
			})
		}
	}
	return hints
}

var formatOnInputRE = regexp.MustCompile(`(?i)(request|input|user|params?\w*).*\.\s*format\s*\(`)

func checkFormatOnInput(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		if formatOnInputRE.MatchString(line) {
			hints = append(hints, Hint{
				Severity: "medium",
				Pattern:  "format-on-input",
				Line:     i + 1,
				Snippet:  strings.TrimSpace(line),
				Message:  ".format() called on user input: potential injection risk. Use sandboxed templates or whitelist placeholders.",
			})
		}
	}
	return hints
}

var moduleMutableRE = regexp.MustCompile(`^(\w+)\s*=\s*(\[\s*\]|\{\s*\})`)

func checkModuleMutableState(lines []string) []Hint {
	var hints []Hint
	mutableVars := make(map[string]int)
	inAsync := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "async def ") {
			inAsync = true
			continue
		}
		if inAsync && strings.HasPrefix(trimmed, "def ") {
			inAsync = false
			continue
		}
		if !strings.HasPrefix(line, " ") && !strings.HasPrefix(line, "\t") {
			if matches := moduleMutableRE.FindStringSubmatch(trimmed); matches != nil {
				mutableVars[matches[1]] = i + 1
			}
		}
		if inAsync && trimmed != "" {
			for varName, varLine := range mutableVars {
				if strings.Contains(trimmed, varName+".") || strings.Contains(trimmed, varName+"[") {
					hints = append(hints, Hint{
						Severity: "low",
						Pattern:  "module-mutable-state",
						Line:     varLine,
						Snippet:  varName,
						Message:  "Module-level mutable state " + varName + " modified in async function: risk of race conditions",
					})
				}
			}
		}
	}
	return hints
}

// stripDiff removes leading diff marker (+ or -) and surrounding whitespace.
func stripDiff(s string) string {
	s = strings.TrimSpace(s)
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		s = strings.TrimSpace(s[1:])
	}
	return s
}

// FormatHints renders analysis hints as a compact text block for prompt injection.
func FormatHints(hints []Hint) string {
	if len(hints) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("<analysis_hints>\n")
	for _, h := range hints {
		b.WriteString("  [")
		b.WriteString(h.Severity)
		b.WriteString("] ")
		b.WriteString(h.Pattern)
		b.WriteString(": line ")
		b.WriteString(formatInt(h.Line))
		b.WriteString(" — ")
		b.WriteString(h.Message)
		b.WriteString("\n")
	}
	b.WriteString("</analysis_hints>")
	return b.String()
}

func formatInt(n int) string {
	if n <= 0 {
		return "?"
	}
	// Simple int-to-string without importing strconv
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
