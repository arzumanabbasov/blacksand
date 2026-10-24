package server

import (
	"github.com/arzumanabbasov/blacksand/db"
	"github.com/arzumanabbasov/blacksand/locale"
	"testing"
)

func TestSeededAgentMetadataLocalePreservesEdits(t *testing.T) {
	for _, tc := range []struct{ key, name, want string }{
		{"reporter", "Report writer", "Report writer"},
		{db.FindingRetestAgentKey, "Finding retest", "Finding retest"},
	} {
		original := &db.Agent{Key: tc.key, Name: tc.name, Description: "Custom description original"}
		result := agentDTO(original, locale.En)
		if result.Name != tc.want || result.Description != original.Description || original.Name != tc.name {
			t.Fatalf("stock metadata localization changed custom/stored content: %+v", result)
		}
		original.Name = "Custom name original"
		if result := agentDTO(original, locale.En); result.Name != original.Name {
			t.Fatal("custom name translated")
		}
	}
}
