package codex

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/llm/transformer/openai/responses"
	"github.com/looplj/axonhub/llm/transformer/shared"
)

// Exercise the HTTP input -> Responses inbound -> Codex outbound chain. The
// changing context sessions reproduce the gateway's per-request fallback IDs.
func TestCodexOutbound_ClientCacheIdentity(t *testing.T) {
	sim := newCodexSimulator(t)
	sim.Inbound = responses.NewInboundTransformer()
	send := func(t *testing.T, scope, requestSession, cacheKey string, headers http.Header) *http.Request {
		t.Helper()
		body, err := json.Marshal(map[string]any{
			"model": "gpt-6-luna", "input": []map[string]string{{"role": "user", "content": "Reply OK"}},
			"prompt_cache_key": cacheKey, "stream": true, "store": false,
		})
		require.NoError(t, err)
		inbound, err := http.NewRequest(http.MethodPost, "http://localhost/v1/responses", bytes.NewReader(body))
		require.NoError(t, err)
		inbound.Header = headers.Clone()
		if inbound.Header == nil {
			inbound.Header = make(http.Header)
		}
		inbound.Header.Set("Content-Type", "application/json")
		ctx := shared.WithSessionScope(shared.WithSessionID(context.Background(), requestSession), scope)
		outbound, err := sim.Simulate(ctx, inbound)
		require.NoError(t, err)
		t.Cleanup(func() { _ = outbound.Body.Close() })
		outboundBody, err := io.ReadAll(outbound.Body)
		require.NoError(t, err)
		var payload map[string]any
		require.NoError(t, json.Unmarshal(outboundBody, &payload))
		assert.Equal(t, cacheKey, payload["prompt_cache_key"])
		return outbound
	}

	t.Run("same caller and cache key keep a stable session", func(t *testing.T) {
		first := send(t, "api_key:3:project:1", "responses-first", "conversation-a", nil)
		second := send(t, "api_key:3:project:1", "responses-second", "conversation-a", nil)
		identity := first.Header.Get("Session-Id")
		require.NotEmpty(t, identity)
		assert.Equal(t, identity, second.Header.Get("Session-Id"))
		for _, req := range []*http.Request{first, second} {
			assert.Equal(t, identity, req.Header.Get("Thread-Id"))
			assert.Equal(t, identity, req.Header.Get("Conversation_id"))
			var metadata TurnMetadata
			require.NoError(t, json.Unmarshal([]byte(req.Header.Get("X-Codex-Turn-Metadata")), &metadata))
			assert.Equal(t, identity, metadata.SessionID)
			assert.Equal(t, identity, metadata.ThreadID)
		}
	})

	t.Run("different callers or cache keys remain separate", func(t *testing.T) {
		first := send(t, "api_key:3:project:1", "responses-first", "conversation-a", nil)
		otherCaller := send(t, "api_key:4:project:1", "responses-second", "conversation-a", nil)
		otherConversation := send(t, "api_key:3:project:1", "responses-third", "conversation-b", nil)
		assert.NotEqual(t, first.Header.Get("Session-Id"), otherCaller.Header.Get("Session-Id"))
		assert.NotEqual(t, first.Header.Get("Session-Id"), otherConversation.Header.Get("Session-Id"))
	})

	t.Run("explicit client identity wins over cache fallback", func(t *testing.T) {
		for _, headers := range []http.Header{
			{"Session-Id": {"client-session"}},
			{"Session_id": {"client-session"}},
			{"X-Codex-Turn-Metadata": {`{"session_id":"client-session","turn_id":"client-turn"}`}},
		} {
			outbound := send(t, "api_key:3:project:1", "responses-first", "conversation-a", headers)
			assert.Equal(t, "client-session", outbound.Header.Get("Session-Id"))
		}
	})

	t.Run("blank cache key preserves existing session fallback", func(t *testing.T) {
		outbound := send(t, "api_key:3:project:1", "existing-session", " ", nil)
		assert.Equal(t, "existing-session", outbound.Header.Get("Session-Id"))
	})

	t.Run("compact preserves the same cache identity", func(t *testing.T) {
		response := send(t, "api_key:3:project:1", "responses-first", "conversation-a", nil)
		sim.Inbound = responses.NewCompactInboundTransformer()
		compact := send(t, "api_key:3:project:1", "compact-second", "conversation-a", nil)
		assert.Equal(t, response.Header.Get("Session-Id"), compact.Header.Get("Session-Id"))
	})
}
