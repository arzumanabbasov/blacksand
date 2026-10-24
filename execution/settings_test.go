package execution

import "testing"

func TestRuntimeSettings(t *testing.T) {
	configMu.Lock()
	previous := configured
	configured = nil
	configMu.Unlock()
	t.Cleanup(func() { configMu.Lock(); configured = previous; configMu.Unlock() })
	t.Setenv("BLACKSAND_EXECUTION", "e2b")
	t.Setenv("E2B_API_KEY", "environment-key")
	if CurrentConfig().APIKey != "environment-key" {
		t.Fatal("environment fallback missing")
	}
	Configure(Config{Mode: "e2b", APIKey: "saved-key", Template: "base"})
	if c := CurrentConfig(); c.APIKey != "saved-key" || !Enabled() {
		t.Fatal("saved settings did not take effect")
	}
}
