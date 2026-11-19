package docs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSwaggerJSON(t *testing.T) {
	raw := GetSwaggerJSON("", "")
	assert.Contains(t, raw, `"host": "localhost:8081"`)
	assert.Contains(t, raw, `"schemes": ["http"]`)

	var parsed map[string]any
	err := json.Unmarshal([]byte(raw), &parsed)
	assert.NoError(t, err)
	assert.Equal(t, "2.0", parsed["swagger"])

	custom := GetSwaggerJSON("api.delcom.org", "https")
	assert.Contains(t, custom, `"host": "api.delcom.org"`)
	assert.Contains(t, custom, `"schemes": ["https"]`)
}
