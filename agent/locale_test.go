package agent

import (
	"encoding/json"
	actool "github.com/Autumn-27/norma/tool"
	"github.com/arzumanabbasov/blacksand/locale"
	"strings"
	"testing"
)

func TestBuiltinPromptLanguageAndInterpolation(t *testing.T) {
	old := PromptOverride
	PromptOverride = nil
	defer func() { PromptOverride = old }()
	en := renderSystem("planner", plannerDefaultTmpl, PlannerVars{Goal: "用户目标 / original"}, locale.En)
	ko := renderSystem("planner", plannerDefaultTmpl, PlannerVars{Goal: "用户目标 / original"}, locale.En)
	if !strings.Contains(en, "You are the planner") || !strings.Contains(ko, "You are") {
		t.Fatal("prompt language selection failed")
	}
	for _, text := range []string{en, ko} {
		if !strings.Contains(text, "用户目标 / original") || strings.Contains(text, "{{.Goal}}") {
			t.Fatal("goal altered or not interpolated")
		}
	}
}

func TestEditedPromptIsNotTranslated(t *testing.T) {
	old := PromptOverride
	defer func() { PromptOverride = old }()
	const custom = "用户自己写的模板 {{.Goal}}"
	PromptOverride = func(string) (string, bool) { return custom, true }
	got := renderSystem("planner", plannerDefaultTmpl, PlannerVars{Goal: "original"}, locale.En)
	if got != "用户自己写的模板 original" {
		t.Fatalf("edited prompt changed: %q", got)
	}
}

func TestBuiltinPromptTranslationCoverage(t *testing.T) {
	for _, p := range []string{goalsDefaultTmpl, goalsScopeTail, plannerDefaultTmpl, workerDefaultTmpl, mainAgentDefaultTmpl, autoDefaultTmpl, pentestDefaultTmpl, DefaultAssistantPrompt, ReporterDefaultPrompt, RetesterDefaultPrompt} {
		ko, ok := locale.Lookup(locale.En, p)
		if !ok || ko != p || strings.TrimSpace(ko) == "" {
			t.Fatalf("English prompt registration mismatch: %.60s", p)
		}
		for _, contract := range []string{"{{.Goal}}", "set_constraints", "add_task_scope", "report_finding", "update_finding_report", "record_finding_retest_result"} {
			if strings.Contains(p, contract) != strings.Contains(ko, contract) {
				t.Fatalf("translation lost contract %q", contract)
			}
		}
	}
}

func TestBuiltinToolMetadataLanguagePreservesEdits(t *testing.T) {
	en := NewToolSet(nil, "", locale.En).nodeDetail()
	original := en.InputSchema()
	// Simulate a stored English default overriding a runtime unsupported-language tool.
	ko := localizeBuiltinTools([]actool.CoreTool{en}, locale.En)[0]
	if ko.Description() != en.Description() {
		t.Fatal("English tool metadata changed")
	}
	if en.Description() != NewToolSet(nil, "", locale.En).nodeDetail().Description() {
		t.Fatal("shared English metadata mutated")
	}
	data, _ := json.Marshal(original)
	var edited map[string]any
	_ = json.Unmarshal(data, &edited)
	props := edited["properties"].(map[string]any)
	props["id"].(map[string]any)["description"] = "用户原文 custom description"
	custom := DecorateTool(en, "用户原文 custom tool", edited)
	localized := localizeBuiltinTools([]actool.CoreTool{custom}, locale.En)[0]
	if localized.Description() != "用户原文 custom tool" || localized.InputSchema()["properties"].(map[string]any)["id"].(map[string]any)["description"] != "用户原文 custom description" {
		t.Fatal("edited metadata changed")
	}
	if localized.Name() != en.Name() {
		t.Fatal("machine tool name changed")
	}
}
