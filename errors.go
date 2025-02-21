package rulesengine

import (
	"net/http"
)

type RulesEngineError struct {
	err    string
	status int
}

func (m RulesEngineError) Error() string {
	return m.err
}

func (m RulesEngineError) StatusCode() int {
	if m.status == 0 {
		return http.StatusInternalServerError
	}
	return m.status
}

func newRulesEngineError(err string, status int) error {
	return RulesEngineError{err: err, status: status}
}

var ErrorUnexpected = newRulesEngineError("unexpected error", http.StatusInternalServerError)
var ErrorFlagNotFound = newRulesEngineError("flag not found", http.StatusNotFound)
