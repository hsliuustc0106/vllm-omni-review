package analysis

import (
	"regexp"
	"strings"
)

// AnalyzeTensorShapes scans for tensor shape/dtype patterns relevant to
// multimodal model code.
func AnalyzeTensorShapes(diff string) []Hint {
	var hints []Hint
	lines := strings.Split(diff, "\n")

	hints = append(hints, checkShapeOps(lines)...)
	hints = append(hints, checkMissingContiguous(lines)...)
	hints = append(hints, checkDtypeConversions(lines)...)
	hints = append(hints, checkConcatOps(lines)...)

	return hints
}

var shapeOpRE = regexp.MustCompile(`\.(view|reshape|permute|transpose|squeeze|unsqueeze|expand)\s*\(`)

func checkShapeOps(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if shapeOpRE.MatchString(trimmed) {
			hints = append(hints, Hint{
				Severity: "low",
				Pattern:  "shape-op",
				Line:     i + 1,
				Snippet:  trimmed,
				Message:  "Tensor shape operation: verify dimension consistency across modalities",
			})
		}
	}
	return hints
}

var contiguousRequiredRE = regexp.MustCompile(`\.(view|reshape|permute|transpose)\s*\(`)

func checkMissingContiguous(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if contiguousRequiredRE.MatchString(trimmed) {
			found := false
			for j := i - 1; j >= 0 && j >= i-3; j-- {
				if strings.Contains(strings.TrimSpace(lines[j]), ".contiguous()") {
					found = true
					break
				}
			}
			if !found {
				hints = append(hints, Hint{
					Severity: "low",
					Pattern:  "missing-contiguous",
					Line:     i + 1,
					Snippet:  trimmed,
					Message:  "Potential missing .contiguous() before shape-changing op",
				})
			}
		}
	}
	return hints
}

var dtypeConvRE = regexp.MustCompile(`\.(half|bfloat16|float|double|to)\s*\(\s*(dtype\s*=|)`)
var toDtypeRE = regexp.MustCompile(`\.to\(\s*.*dtype\s*=`)

func checkDtypeConversions(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if dtypeConvRE.MatchString(trimmed) || toDtypeRE.MatchString(trimmed) {
			hints = append(hints, Hint{
				Severity: "low",
				Pattern:  "dtype-conversion",
				Line:     i + 1,
				Snippet:  trimmed,
				Message:  "Dtype conversion: verify mixed-precision correctness at fusion points",
			})
		}
	}
	return hints
}

var concatRE = regexp.MustCompile(`torch\.(cat|stack)\s*\(`)

func checkConcatOps(lines []string) []Hint {
	var hints []Hint
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if concatRE.MatchString(trimmed) {
			hints = append(hints, Hint{
				Severity: "low",
				Pattern:  "concat-op",
				Line:     i + 1,
				Snippet:  trimmed,
				Message:  "Tensor concat/stack: verify dimension compatibility across tensors",
			})
		}
	}
	return hints
}
