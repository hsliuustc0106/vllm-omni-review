package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandBraces_NoBraces(t *testing.T) {
	got := expandBraces("*.java")
	if len(got) != 1 || got[0] != "*.java" {
		t.Errorf("expected [*.java], got %v", got)
	}
}

func TestExpandBraces_SingleGroup(t *testing.T) {
	got := expandBraces("*.{go,py}")
	want := []string{"*.go", "*.py"}
	if len(got) != len(want) {
		t.Fatalf("expected %d items, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}

func TestExpandBraces_MultipleOptions(t *testing.T) {
	got := expandBraces("**/*.{ts,js,tsx,jsx}")
	want := []string{"**/*.ts", "**/*.js", "**/*.tsx", "**/*.jsx"}
	if len(got) != len(want) {
		t.Fatalf("expected %d items, got %d: %v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}

func TestExpandBraces_UnclosedBrace(t *testing.T) {
	got := expandBraces("*.{go,py")
	if len(got) != 1 || got[0] != "*.{go,py" {
		t.Errorf("expected original pattern, got %v", got)
	}
}

func TestResolve_DefaultRules(t *testing.T) {
	rule, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault: %v", err)
	}

	tests := []struct {
		path       string
		wantSubstr string
	}{
		{"vllm_omni/engine/scheduler.py", "Scheduler Correctness"},
		{"vllm_omni/model_executor/models/qwen.py", "Model Registration"},
		{"vllm_omni/diffusion/models/z_image.py", "Latent Cache Lifecycle"},
		{"vllm_omni/entrypoints/openai/api.py", "Input Validation"},
		{"vllm_omni/connectors/shm.py", "Resource Management"},
		{"vllm_omni/distributed/omni_connectors/utils.py", "Resource Management"},
		{"vllm_omni/platforms/cuda/attention.py", "Device-Specific Code"},
		{"vllm_omni/quantization/fp8.py", "Weight Packing Correctness"},
		{"vllm_omni/config/stage_config.yaml", "Validation"},
		{"vllm_omni/utils/helpers.py", "Correctness"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := rule.Resolve(tt.path)
			if !strings.Contains(got, tt.wantSubstr) {
				t.Errorf("Resolve(%q): expected rule containing %q, got %q",
					tt.path, tt.wantSubstr, truncate(got, 80))
			}
		})
	}
}

func TestResolve_FallbackToDefault(t *testing.T) {
	rule, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault: %v", err)
	}

	paths := []string{
		"readme.md",
		"docs/architecture.txt",
		"Makefile",
		"src/lib.rs",
		"scripts/helper.sh",
	}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			got := rule.Resolve(path)
			if got != rule.DefaultRule {
				t.Errorf("Resolve(%q): expected DefaultRule, got %q", path, truncate(got, 80))
			}
		})
	}
}

func TestResolve_CustomRule_FirstMatchWins(t *testing.T) {
	rule := &SystemRule{
		DefaultRule: "default",
		PathRules: []PathRule{
			{Pattern: "**/engine/**/*.py", Rule: "engine-rule"},
			{Pattern: "**/*.py", Rule: "python-rule"},
		},
	}

	got := rule.Resolve("vllm_omni/engine/scheduler.py")
	if got != "engine-rule" {
		t.Errorf("expected engine-rule, got %q", got)
	}

	got = rule.Resolve("vllm_omni/utils/helpers.py")
	if got != "python-rule" {
		t.Errorf("expected python-rule, got %q", got)
	}
}

func TestResolve_CustomRule_DefaultFallback(t *testing.T) {
	rule := &SystemRule{
		DefaultRule: "fallback-rule",
		PathRules: []PathRule{
			{Pattern: "**/*.py", Rule: "python-rule"},
		},
	}

	got := rule.Resolve("main.go")
	if got != "fallback-rule" {
		t.Errorf("expected fallback-rule, got %q", got)
	}
}

func TestResolve_CaseSensitivity(t *testing.T) {
	rule := &SystemRule{
		DefaultRule: "default",
		PathRules: []PathRule{
			{Pattern: "**/*.py", Rule: "python-rule"},
		},
	}

	got := rule.Resolve("Foo.PY")
	if got != "default" {
		t.Errorf("expected default for uppercase extension, got %q", got)
	}

	got = rule.Resolve("foo.py")
	if got != "python-rule" {
		t.Errorf("expected python-rule for lowercase, got %q", got)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func TestNewResolver_DefaultOnly(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resolver, _, err := NewResolver(t.TempDir(), "")

	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	got := resolver.Resolve("vllm_omni/engine/scheduler.py")
	if !strings.Contains(got, "Scheduler Correctness") {
		t.Errorf("expected engine rule, got %q", truncate(got, 80))
	}
}

func TestNewResolver_ProjectFileMissing(t *testing.T) {
	resolver, _, err := NewResolver(t.TempDir(), "")

	if err != nil {
		t.Fatalf("NewResolver should not fail when project rule is missing: %v", err)
	}
	got := resolver.Resolve("readme.md")
	if got == "" {
		t.Errorf("expected non-empty default rule")
	}
}

func TestNewResolver_ProjectRuleHighestPriority(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	ocrDir := filepath.Join(dir, ".opencodereview")
	if err := os.MkdirAll(ocrDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ruleJSON := `{"rules":[{"path":"vllm_omni/engine/**/*.py","rule":"project-engine-rule"}]}`
	if err := os.WriteFile(filepath.Join(ocrDir, "rule.json"), []byte(ruleJSON), 0o644); err != nil {
		t.Fatalf("write rule.json: %v", err)
	}

	resolver, _, err := NewResolver(dir, "")
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	tests := []struct {
		path string
		want string
	}{
		{"vllm_omni/engine/scheduler.py", "project-engine-rule"},
		{"vllm_omni/utils/helpers.py", "Correctness"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := resolver.Resolve(tt.path)
			if !strings.Contains(got, tt.want) {
				t.Errorf("Resolve(%q) = %q, want containing %q", tt.path, truncate(got, 80), tt.want)
			}
		})
	}
}

func TestNewResolver_ProjectRuleFallsBackToSystem(t *testing.T) {
	dir := t.TempDir()
	ocrDir := filepath.Join(dir, ".opencodereview")
	if err := os.MkdirAll(ocrDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ruleJSON := `{"rules":[{"path":"special/**/*.py","rule":"special-py-rule"}]}`
	if err := os.WriteFile(filepath.Join(ocrDir, "rule.json"), []byte(ruleJSON), 0o644); err != nil {
		t.Fatalf("write rule.json: %v", err)
	}

	resolver, _, err := NewResolver(dir, "")
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	got := resolver.Resolve("vllm_omni/utils/helpers.py")
	if !strings.Contains(got, "Correctness") {
		t.Errorf("expected system default rule, got %q", truncate(got, 80))
	}
}

func TestNewResolver_CustomRuleOverridesDefault(t *testing.T) {
	dir := t.TempDir()
	customRule := `{"rules":[{"path":"**/*.py","rule":"custom-py-rule"}]}`
	customPath := filepath.Join(dir, "custom_rules.json")
	if err := os.WriteFile(customPath, []byte(customRule), 0o644); err != nil {
		t.Fatalf("write custom rule: %v", err)
	}

	resolver, _, err := NewResolver(t.TempDir(), customPath)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	got := resolver.Resolve("vllm_omni/engine/scheduler.py")
	if got != "custom-py-rule" {
		t.Errorf("expected custom-py-rule, got %q", got)
	}
	got = resolver.Resolve("readme.md")
	if !strings.Contains(got, "Correctness") {
		t.Errorf("expected system default rule, got %q", truncate(got, 80))
	}
}

func TestNewResolver_CustomOverridesProject(t *testing.T) {
	customDir := t.TempDir()
	customRule := `{"rules":[{"path":"**/*.py","rule":"custom-py-rule"}]}`
	customPath := filepath.Join(customDir, "custom_rules.json")
	if err := os.WriteFile(customPath, []byte(customRule), 0o644); err != nil {
		t.Fatalf("write custom rule: %v", err)
	}

	repoDir := t.TempDir()
	ocrDir := filepath.Join(repoDir, ".opencodereview")
	if err := os.MkdirAll(ocrDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	projRule := `{"rules":[{"path":"vllm_omni/engine/**/*.py","rule":"project-engine-rule"},{"path":"**/*.go","rule":"project-go-rule"}]}`
	if err := os.WriteFile(filepath.Join(ocrDir, "rule.json"), []byte(projRule), 0o644); err != nil {
		t.Fatalf("write rule.json: %v", err)
	}

	resolver, _, err := NewResolver(repoDir, customPath)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	tests := []struct {
		path string
		want string
	}{
		{"vllm_omni/engine/scheduler.py", "custom-py-rule"},
		{"vllm_omni/utils/helpers.py", "custom-py-rule"},
		{"main.go", "project-go-rule"},
		{"readme.md", "Correctness"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := resolver.Resolve(tt.path)
			if !strings.Contains(got, tt.want) {
				t.Errorf("Resolve(%q) = %q, want containing %q", tt.path, truncate(got, 80), tt.want)
			}
		})
	}
}

func TestNewResolver_ProjectFileMalformed(t *testing.T) {
	dir := t.TempDir()
	ocrDir := filepath.Join(dir, ".opencodereview")
	if err := os.MkdirAll(ocrDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(ocrDir, "rule.json"), []byte("{invalid json"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, _, err := NewResolver(dir, "")
	if err == nil {
		t.Errorf("expected error for malformed project rule.json")
	}
}

func TestFileFilter_IsUserExcluded(t *testing.T) {
	f := &FileFilter{
		Exclude: []string{"**/generated/**", "**/*.pb.go", "vendor/**/*.{go,js}"},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"src/generated/api.py", true},
		{"pkg/foo.pb.go", true},
		{"vendor/lib/util.go", true},
		{"vendor/lib/util.js", true},
		{"src/main.go", false},
		{"src/generated.go", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := f.IsUserExcluded(tt.path); got != tt.want {
				t.Errorf("IsUserExcluded(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFileFilter_IsUserIncluded(t *testing.T) {
	f := &FileFilter{
		Include: []string{"src/**/*.py", "src/**/*.{cu,cuh}"},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"src/main/foo.py", true},
		{"src/main/bar.cu", true},
		{"src/build.cuh", true},
		{"test/main.py", false},
		{"src/main/util.go", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := f.IsUserIncluded(tt.path); got != tt.want {
				t.Errorf("IsUserIncluded(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestFileFilter_IsUserIncluded_EmptyInclude(t *testing.T) {
	f := &FileFilter{}
	if f.IsUserIncluded("anything.py") {
		t.Errorf("expected false when include is empty")
	}
}

func TestFileFilter_CaseInsensitive(t *testing.T) {
	f := &FileFilter{
		Include: []string{"src/**/*.py"},
		Exclude: []string{"**/generated/**"},
	}

	if !f.IsUserIncluded("SRC/Main/Foo.PY") {
		t.Errorf("expected case-insensitive include match")
	}
	if !f.IsUserExcluded("SRC/Generated/Api.py") {
		t.Errorf("expected case-insensitive exclude match")
	}
}

func TestNewResolver_FileFilterMerged(t *testing.T) {
	repoDir := t.TempDir()
	ocrDir := filepath.Join(repoDir, ".opencodereview")
	if err := os.MkdirAll(ocrDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	projJSON := `{"rules":[],"include":["src/**/*.py"],"exclude":["**/generated/**"]}`
	if err := os.WriteFile(filepath.Join(ocrDir, "rule.json"), []byte(projJSON), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, filter, err := NewResolver(repoDir, "")
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	if filter == nil {
		t.Fatal("expected non-nil FileFilter")
	}
	if !filter.HasInclude() {
		t.Error("expected HasInclude to be true")
	}
	if !filter.IsUserIncluded("src/main/foo.py") {
		t.Error("expected src/main/foo.py to be included")
	}
	if !filter.IsUserExcluded("src/generated/api.py") {
		t.Error("expected src/generated/api.py to be excluded")
	}
}

func TestNewResolver_FileFilterNilWhenEmpty(t *testing.T) {
	_, filter, err := NewResolver(t.TempDir(), "")
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	if filter != nil {
		t.Errorf("expected nil FileFilter when no include/exclude configured, got %+v", filter)
	}
}

func TestNewResolver_FileFilterPriorityOverride(t *testing.T) {
	repoDir := t.TempDir()
	ocrDir := filepath.Join(repoDir, ".opencodereview")
	if err := os.MkdirAll(ocrDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	projJSON := `{"rules":[],"include":["src/**/*.py"],"exclude":["**/gen/**"]}`
	if err := os.WriteFile(filepath.Join(ocrDir, "rule.json"), []byte(projJSON), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	customDir := t.TempDir()
	customJSON := `{"rules":[],"include":["lib/**/*.py"],"exclude":["**/tmp/**"]}`
	customPath := filepath.Join(customDir, "custom.json")
	if err := os.WriteFile(customPath, []byte(customJSON), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, filter, err := NewResolver(repoDir, customPath)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	if filter == nil {
		t.Fatal("expected non-nil FileFilter")
	}

	if !filter.IsUserIncluded("lib/util.py") {
		t.Error("expected custom include to be active")
	}
	if !filter.IsUserExcluded("lib/tmp/cache.py") {
		t.Error("expected custom exclude to be active")
	}
	if filter.IsUserIncluded("src/main/foo.py") {
		t.Error("project include should not be active when custom is present")
	}
	if filter.IsUserExcluded("src/gen/api.py") {
		t.Error("project exclude should not be active when custom is present")
	}
}

func TestNewResolver_FileFilterFallsToProject(t *testing.T) {
	repoDir := t.TempDir()
	ocrDir := filepath.Join(repoDir, ".opencodereview")
	if err := os.MkdirAll(ocrDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	projJSON := `{"rules":[],"include":["src/**/*.py"],"exclude":["**/gen/**"]}`
	if err := os.WriteFile(filepath.Join(ocrDir, "rule.json"), []byte(projJSON), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	customDir := t.TempDir()
	customJSON := `{"rules":[{"path":"**/*.go","rule":"custom-go"}]}`
	customPath := filepath.Join(customDir, "custom.json")
	if err := os.WriteFile(customPath, []byte(customJSON), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, filter, err := NewResolver(repoDir, customPath)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	if filter == nil {
		t.Fatal("expected non-nil FileFilter from project layer")
	}
	if !filter.IsUserIncluded("src/main/foo.py") {
		t.Error("expected project include to take effect when custom has none")
	}
}

func TestResolveDetail_SystemDefault(t *testing.T) {
	resolver, _, err := NewResolver(t.TempDir(), "")
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	dr := resolver.(DetailResolver)

	detail := dr.ResolveDetail("readme.md")
	if detail.Source != "system" {
		t.Errorf("expected source 'system', got %q", detail.Source)
	}
	if detail.Pattern != "default" {
		t.Errorf("expected pattern 'default', got %q", detail.Pattern)
	}
	if !strings.Contains(detail.Rule, "Correctness") {
		t.Errorf("expected default rule content, got %q", truncate(detail.Rule, 80))
	}
}

func TestResolveDetail_SystemPatternMatch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	resolver, _, err := NewResolver(t.TempDir(), "")
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	dr := resolver.(DetailResolver)

	detail := dr.ResolveDetail("vllm_omni/engine/scheduler.py")
	if detail.Source != "system" {
		t.Errorf("expected source 'system', got %q", detail.Source)
	}
	if detail.Pattern != "**/engine/**/*.py" {
		t.Errorf("expected pattern '**/engine/**/*.py', got %q", detail.Pattern)
	}
	if !strings.Contains(detail.Rule, "Scheduler Correctness") {
		t.Errorf("expected engine rule, got %q", truncate(detail.Rule, 80))
	}
}

func TestResolveDetail_ProjectOverridesSystem(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	ocrDir := filepath.Join(dir, ".opencodereview")
	if err := os.MkdirAll(ocrDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ruleJSON := `{"rules":[{"path":"vllm_omni/engine/**/*.py","rule":"project-engine-rule"}]}`
	if err := os.WriteFile(filepath.Join(ocrDir, "rule.json"), []byte(ruleJSON), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	resolver, _, err := NewResolver(dir, "")
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	dr := resolver.(DetailResolver)

	detail := dr.ResolveDetail("vllm_omni/engine/scheduler.py")
	if detail.Source != "project" {
		t.Errorf("expected source 'project', got %q", detail.Source)
	}
	if detail.Pattern != "vllm_omni/engine/**/*.py" {
		t.Errorf("expected pattern, got %q", detail.Pattern)
	}
	if detail.Rule != "project-engine-rule" {
		t.Errorf("expected 'project-engine-rule', got %q", detail.Rule)
	}

	detail = dr.ResolveDetail("vllm_omni/utils/helpers.py")
	if detail.Source != "system" {
		t.Errorf("expected source 'system', got %q", detail.Source)
	}
}

func TestResolveDetail_CustomOverridesAll(t *testing.T) {
	repoDir := t.TempDir()
	ocrDir := filepath.Join(repoDir, ".opencodereview")
	if err := os.MkdirAll(ocrDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	projJSON := `{"rules":[{"path":"**/*.py","rule":"project-py-rule"}]}`
	if err := os.WriteFile(filepath.Join(ocrDir, "rule.json"), []byte(projJSON), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	customDir := t.TempDir()
	customJSON := `{"rules":[{"path":"**/*.py","rule":"custom-py-rule"}]}`
	customPath := filepath.Join(customDir, "custom.json")
	if err := os.WriteFile(customPath, []byte(customJSON), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	resolver, _, err := NewResolver(repoDir, customPath)
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}
	dr := resolver.(DetailResolver)

	detail := dr.ResolveDetail("vllm_omni/engine/scheduler.py")
	if detail.Source != "custom" {
		t.Errorf("expected source 'custom', got %q", detail.Source)
	}
	if detail.Rule != "custom-py-rule" {
		t.Errorf("expected 'custom-py-rule', got %q", detail.Rule)
	}
}

func TestNewResolver_BraceExpansionInProjectRule(t *testing.T) {
	dir := t.TempDir()
	ocrDir := filepath.Join(dir, ".opencodereview")
	if err := os.MkdirAll(ocrDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	ruleJSON := `{"rules":[{"path":"src/**/*.{py,pyi}","rule":"py-rule"}]}`
	if err := os.WriteFile(filepath.Join(ocrDir, "rule.json"), []byte(ruleJSON), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	resolver, _, err := NewResolver(dir, "")
	if err != nil {
		t.Fatalf("NewResolver: %v", err)
	}

	tests := []struct {
		path string
		want string
	}{
		{"src/main/foo.py", "py-rule"},
		{"src/main/bar.pyi", "py-rule"},
		{"src/main/baz.go", "Correctness"},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := resolver.Resolve(tt.path)
			if !strings.Contains(got, tt.want) {
				t.Errorf("Resolve(%q) = %q, want containing %q", tt.path, truncate(got, 80), tt.want)
			}
		})
	}
}
