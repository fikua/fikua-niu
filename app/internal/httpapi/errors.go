package httpapi

import (
	"encoding/json"
	"net/http"
)

// Error codes/messages introduced by NIU-4 (design.md §6.1). Declared as
// constants so every call site (auth_handlers.go, csrf.go) uses the exact
// same wording — AC-11/S5 require the invalid_credentials body to be
// byte-identical regardless of which code path produced it.
const (
	msgInvalidCredentials = "Usuari o contrasenya incorrectes."
	msgRateLimited        = "Massa intents. Torna-ho a provar més tard."
	msgCSRFFailed         = "Petició no vàlida."
	msgLoginValidation    = "Cal indicar usuari i contrasenya."
)

// apiError is the uniform error envelope (design.md §6.1, PLAN.md §2.5):
// {"error": {"code": "...", "message": "..."}}. Never carries internal
// error detail (SQL, stack traces, file paths) — those are logged
// server-side only.
type apiError struct {
	Error apiErrorBody `json:"error"`
}

type apiErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError{Error: apiErrorBody{Code: code, Message: message}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
