package biz

import (
	"context"
	"encoding/json"
	"slices"
	"time"

	"github.com/looplj/axonhub/internal/contexts"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/log"
)

// SystemKeyRequestPayloadCapture is independent of the global storage policy,
// so saving the policy in the admin UI does not remove a temporary capture.
const SystemKeyRequestPayloadCapture = "request_payload_capture"

type RequestPayloadCapture struct {
	APIKeyIDs []int     `json:"api_key_ids"`
	ExpiresAt time.Time `json:"expires_at"`
}

// applyRequestPayloadCapture operates on a freshly decoded policy, never on a
// cached policy shared between callers. Administrative and GC contexts have no
// API key and continue to see the global policy.
func (s *SystemService) applyRequestPayloadCapture(ctx context.Context, policy *StoragePolicy) *StoragePolicy {
	apiKey, ok := contexts.GetAPIKey(ctx)
	if !ok || apiKey == nil || apiKey.ID <= 0 {
		return policy
	}

	value, err := s.getSystemValue(ctx, SystemKeyRequestPayloadCapture)
	if err != nil {
		if !ent.IsNotFound(err) {
			log.Warn(ctx, "Failed to load request payload capture; keeping global storage policy", log.Cause(err))
		}
		return policy
	}

	var capture RequestPayloadCapture
	if err := json.Unmarshal([]byte(value), &capture); err != nil {
		log.Warn(ctx, "Invalid request payload capture; keeping global storage policy", log.Cause(err))
		return policy
	}
	if !time.Now().Before(capture.ExpiresAt) || !slices.Contains(capture.APIKeyIDs, apiKey.ID) {
		return policy
	}

	policy.StoreRequestBody = true
	policy.StoreResponseBody = true
	policy.StoreChunks = true
	return policy
}
