package server

import (
	"encoding/json"
	"github.com/arzumanabbasov/blacksand/execution"
	"net/http"
	"strings"
)

func (s *Server) getE2BSettings(w http.ResponseWriter, r *http.Request) {
	c := execution.CurrentConfig()
	writeJSON(w, 200, map[string]any{"mode": c.Mode, "key_set": c.APIKey != "", "template": c.Template, "internet_access": true})
}
func (s *Server) putE2BSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		APIKey   *string `json:"api_key"`
		Template *string `json:"template"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16384)
	if json.NewDecoder(r.Body).Decode(&req) != nil {
		writeErr(w, 400, "Invalid E2B settings")
		return
	}
	c := execution.CurrentConfig()
	c.Mode = "e2b"
	if req.APIKey != nil {
		c.APIKey = strings.TrimSpace(*req.APIKey)
	}
	if req.Template != nil {
		c.Template = strings.TrimSpace(*req.Template)
	}
	if strings.ContainsAny(c.APIKey+c.Template, "\x00\r\n") {
		writeErr(w, 400, "Invalid E2B settings")
		return
	}
	if c.APIKey == "" {
		writeErr(w, 400, "Enter an E2B API key")
		return
	}
	if !strings.HasPrefix(c.APIKey, "e2b_") {
		writeErr(w, 400, "Invalid E2B API key: use a key starting with e2b_ from the E2B dashboard")
		return
	}
	raw, _ := json.Marshal(c)
	if err := s.m.pg.SetSetting("e2b_execution", string(raw)); err != nil {
		writeError(w, 500, err)
		return
	}
	execution.Configure(c)
	s.getE2BSettings(w, r)
}
