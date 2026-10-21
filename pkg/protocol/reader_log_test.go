package protocol

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/logtest"
)

func TestMessageReader_UnparseableLineWarningOmitsLineContent(t *testing.T) {
	const secret = "hunter2-reader-secret"

	var logs logtest.Recorder

	reader := NewMessageReader(strings.NewReader("password="+secret+"\n"), SyncMessageMaxSize)
	reader.logger = logs.Logger()

	_, err := reader.Read()
	require.True(t, errors.Is(err, io.EOF))

	output := logs.String()
	require.Contains(t, output, "skipping unparseable line")
	assert.NotContains(t, output, secret)
}
