package analysis

import (
	"strings"
	"testing"
)

func TestSilentSwallow(t *testing.T) {
	diff := "+try:\n+    result = risky_operation()\n+except:\n+    pass"
	hints := AnalyzePython(diff)
	found := false
	for _, h := range hints {
		if h.Pattern == "silent-swallow" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected silent-swallow hint")
	}
}

func TestHardcodedSecret(t *testing.T) {
	diff := `+API_KEY = "sk-abcdefghijklmnopqrstuvwxyz123456"`
	hints := AnalyzePython(diff)
	found := false
	for _, h := range hints {
		if h.Pattern == "hardcoded-secret" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected hardcoded-secret hint")
	}
}

func TestBlockingInAsync(t *testing.T) {
	diff := "+async def process(self, request):\n+    result = requests.get(url)\n+    return result"
	hints := AnalyzePython(diff)
	found := false
	for _, h := range hints {
		if h.Pattern == "blocking-in-async" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected blocking-in-async hint")
	}
}

func TestDangerousCall(t *testing.T) {
	diff := "+result = eval(user_expression)"
	hints := AnalyzePython(diff)
	found := false
	for _, h := range hints {
		if h.Pattern == "dangerous-call" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected dangerous-call hint")
	}
}

func TestMixinMRO(t *testing.T) {
	diff := "+class MyModel(nn.Module, SomeMixin):"
	hints := AnalyzePython(diff)
	found := false
	for _, h := range hints {
		if h.Pattern == "mixin-mro" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected mixin-mro hint")
	}
}

func TestFormatOnInput(t *testing.T) {
	diff := `+msg = request.args.get("template").format(user=current_user)`
	hints := AnalyzePython(diff)
	found := false
	for _, h := range hints {
		if h.Pattern == "format-on-input" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected format-on-input hint")
	}
}

func TestShapeOps(t *testing.T) {
	diff := "+x = x.view(batch, -1, hidden)"
	hints := AnalyzeTensorShapes(diff)
	found := false
	for _, h := range hints {
		if h.Pattern == "shape-op" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected shape-op hint")
	}
}

func TestMissingContiguous(t *testing.T) {
	diff := "+x = x.view(batch, -1, hidden)"
	hints := AnalyzeTensorShapes(diff)
	found := false
	for _, h := range hints {
		if h.Pattern == "missing-contiguous" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected missing-contiguous hint")
	}
}

func TestCustomStream(t *testing.T) {
	diff := "+stream = torch.cuda.Stream()\n+x = some_op(x)"
	hints := AnalyzePlatform(diff)
	found := false
	for _, h := range hints {
		if h.Pattern == "custom-stream-no-sync" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected custom-stream-no-sync hint")
	}
}

func TestFormatHints(t *testing.T) {
	hints := []Hint{
		{Severity: "high", Pattern: "silent-swallow", Line: 42, Message: "Silent exception swallow"},
	}
	result := FormatHints(hints)
	if !strings.Contains(result, "<analysis_hints>") {
		t.Error("expected analysis_hints wrapper")
	}
	if !strings.Contains(result, "silent-swallow") {
		t.Error("expected pattern name in output")
	}
}

func TestEmptyHints(t *testing.T) {
	result := FormatHints(nil)
	if result != "" {
		t.Error("expected empty string for nil hints")
	}
}

func TestNoFalsePositive(t *testing.T) {
	// Normal f-string should not trigger dangerous-call
	diff := `+msg = f"Hello, {name}!"`
	hints := AnalyzePython(diff)
	for _, h := range hints {
		if h.Pattern == "dangerous-call" {
			t.Error("f-string should not trigger dangerous-call")
		}
	}
}

func TestAnalyzeAll(t *testing.T) {
	diff := "+async def process(self, request):\n+    result = requests.get(url)\n+    try:\n+        x = x.view(batch, -1)\n+    except:\n+        pass\n+    return result"
	result := AnalyzeAll(diff)
	if !strings.Contains(result, "blocking-in-async") {
		t.Error("expected blocking-in-async in AnalyzeAll")
	}
	if !strings.Contains(result, "silent-swallow") {
		t.Error("expected silent-swallow in AnalyzeAll")
	}
	if !strings.Contains(result, "shape-op") {
		t.Error("expected shape-op in AnalyzeAll")
	}
}
