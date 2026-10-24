package execution

import (
	"os"
	"sync"
)

type Config struct {
	Mode     string `json:"mode"`
	APIKey   string `json:"api_key"`
	Template string `json:"template"`
}

var configMu sync.RWMutex
var configured *Config

func Configure(c Config) { configMu.Lock(); defer configMu.Unlock(); configured = &c }
func CurrentConfig() Config {
	configMu.RLock()
	defer configMu.RUnlock()
	if configured != nil {
		return *configured
	}
	return Config{Mode: os.Getenv("BLACKSAND_EXECUTION"), APIKey: os.Getenv("E2B_API_KEY"), Template: os.Getenv("BLACKSAND_E2B_TEMPLATE")}
}
