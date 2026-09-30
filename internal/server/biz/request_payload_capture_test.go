package biz

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/pkg/xcache"
)

func TestRequestPayloadCapture(t *testing.T) {
	service, client := setupTestSystemService(t, xcache.Config{Mode: xcache.ModeMemory})
	defer client.Close()
	ctx := authz.WithTestBypass(ent.NewContext(t.Context(), client))
	require.NoError(t, service.SetStoragePolicy(ctx, &StoragePolicy{}))
	keyCtx := contexts.WithAPIKey(ctx, &ent.APIKey{ID: 4})
	policy, err := service.StoragePolicy(keyCtx)
	require.NoError(t, err)
	require.False(t, policy.StoreRequestBody)
	require.False(t, policy.StoreResponseBody)
	require.False(t, policy.StoreChunks)
	capture, err := json.Marshal(RequestPayloadCapture{APIKeyIDs: []int{4}, ExpiresAt: time.Now().Add(time.Hour)})
	require.NoError(t, err)
	require.NoError(t, service.setSystemValue(ctx, SystemKeyRequestPayloadCapture, string(capture)))

	for _, tc := range []struct {
		name string
		id   int
		want bool
	}{
		{"target", 4, true},
		{"other_key", 3, false},
		{"admin_and_gc", 0, false},
		{"target_again", 4, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			callCtx := ctx
			if tc.id != 0 {
				callCtx = contexts.WithAPIKey(ctx, &ent.APIKey{ID: tc.id})
			}
			policy, err := service.StoragePolicy(callCtx)
			require.NoError(t, err)
			require.Equal(t, tc.want, policy.StoreRequestBody)
			require.Equal(t, tc.want, policy.StoreResponseBody)
			require.Equal(t, tc.want, policy.StoreChunks)
			require.False(t, policy.LivePreview)
		})
	}

	for _, value := range []string{
		`{"api_key_ids":[4],"expires_at":"2000-01-01T00:00:00Z"}`,
		`{"api_key_ids":[4]}`,
		`invalid-json`,
	} {
		require.NoError(t, service.setSystemValue(ctx, SystemKeyRequestPayloadCapture, value))
		policy, err := service.StoragePolicy(keyCtx)
		require.NoError(t, err)
		require.False(t, policy.StoreRequestBody)
		require.False(t, policy.StoreResponseBody)
		require.False(t, policy.StoreChunks)
	}
}
