package pipelineconnect

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckServiceAccount(t *testing.T) {
	saved := `{"service_account_name":"saved-sa"}`
	stored := `{"service_account_name":"stored-sa"}`
	cleared := `{}`

	assert.Equal(t, "saved-sa", checkServiceAccount(&saved, &stored))
	assert.Equal(t, "stored-sa", checkServiceAccount(nil, &stored))
	assert.Empty(t, checkServiceAccount(&cleared, &stored))
	assert.Empty(t, checkServiceAccount(nil, nil))
}
