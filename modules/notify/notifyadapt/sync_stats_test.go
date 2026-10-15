package notifyadapt

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSyncStats(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    *parsedStats
		wantErr bool
	}{
		{name: "empty", raw: "", want: &parsedStats{}},
		{name: "empty object", raw: "{}", want: &parsedStats{}},
		{
			name: "nanosecond duration converted to milliseconds",
			raw:  `{"records_read":120,"bytes_synced":2048,"duration":2500000000}`,
			want: &parsedStats{RecordsRead: 120, BytesSynced: 2048, DurationMs: 2500},
		},
		{name: "malformed JSON", raw: `{"records_read":`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSyncStats(tt.raw)
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
