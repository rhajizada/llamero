package main

import (
	"errors"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/rhajizada/llamero/internal/config"
	"github.com/rhajizada/llamero/internal/workers"
)

type fakeScheduler struct {
	registerFn func(string, *asynq.Task, ...asynq.Option) (string, error)
}

func (f fakeScheduler) Register(spec string, task *asynq.Task, opts ...asynq.Option) (string, error) {
	return f.registerFn(spec, task, opts...)
}

func TestNewRedisClientOpt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  config.RedisConfig
	}{
		{
			name: "maps all fields",
			cfg: config.RedisConfig{
				Addr:     "redis:6379",
				Username: "user",
				Password: "secret",
				DB:       3,
			},
		},
		{name: "supports zero values", cfg: config.RedisConfig{}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			opt := NewRedisClientOpt(tc.cfg)
			require.NotNil(t, opt)
			assert.Equal(t, tc.cfg.Addr, opt.Addr)
			assert.Equal(t, tc.cfg.Username, opt.Username)
			assert.Equal(t, tc.cfg.Password, opt.Password)
			assert.Equal(t, tc.cfg.DB, opt.DB)
		})
	}
}

func TestRegisterBackendPingSchedule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		wantErr string
	}{
		{name: "registers sync task"},
		{
			name:    "surfaces register error",
			err:     errors.New("boom"),
			wantErr: "register schedule: boom",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			called := false
			err := RegisterBackendPingSchedule(
				fakeScheduler{registerFn: func(
					spec string,
					task *asynq.Task,
					_ ...asynq.Option,
				) (string, error) {
					called = true
					assert.Equal(t, "@every 5m", spec)
					require.NotNil(t, task)
					assert.Equal(t, workers.TypeSyncBackends, task.Type())
					return "entry-id", tc.err
				}},
				"@every 5m",
			)

			assert.True(t, called)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Equal(t, tc.wantErr, err.Error())
				return
			}

			assert.NoError(t, err)
		})
	}
}
