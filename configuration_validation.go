package main

import "encoding/json"

// Zero is meaningful for retention and budgets; absence must never clear them.
func requireConfigurationFields(body []byte, fields ...string) error {
	var object map[string]json.RawMessage
	if json.Unmarshal(body, &object) != nil || len(object) != len(fields) {
		return fail(400, "invalid_configuration", "complete configuration required")
	}
	for _, field := range fields {
		raw, ok := object[field]
		if !ok || string(raw) == "null" {
			return fail(400, "invalid_configuration", "complete configuration required")
		}
	}
	return nil
}
