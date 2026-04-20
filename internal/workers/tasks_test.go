package workers_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/workers"
)

func TestTaskFactories(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		run     func(*testing.T)
		wantErr bool
	}{
		{
			name: "creates sync backends task",
			run: func(t *testing.T) {
				task, err := workers.NewSyncBackendsTask()
				require.NoError(t, err)
				assert.Equal(t, workers.TypeSyncBackends, task.Type())
				assert.Empty(t, task.Payload())
			},
		},
		{
			name: "creates sync backend by id task",
			run: func(t *testing.T) {
				task, err := workers.NewSyncBackendByIDTask(" backend-1 ")
				require.NoError(t, err)
				assert.Equal(t, workers.TypeSyncBackendByID, task.Type())

				var payload workers.SyncBackendPayload
				require.NoError(t, json.Unmarshal(task.Payload(), &payload))
				assert.Equal(t, "backend-1", payload.BackendID)
			},
		},
		{
			name: "rejects empty backend id",
			run: func(t *testing.T) {
				_, err := workers.NewSyncBackendByIDTask("   ")
				assert.Error(t, err)
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.run(t)
		})
	}
}
