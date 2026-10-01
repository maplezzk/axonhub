# Temporary request payload capture by API key

Use this to investigate prompt caching while keeping the global storage policy unchanged. In the `systems` table, set `request_payload_capture` to a JSON object:

```json
{"api_key_ids":[4],"expires_at":"2026-10-07T16:00:00Z"}
```

The IDs are internal API key IDs, not secret key values. The expiration is required; a missing or elapsed expiration disables the override. Empty `api_key_ids` disables capture. Restart the service after direct database changes so its settings cache is refreshed. Back up the existing value before changing it.

For a matching authenticated request, AxonHub enables its existing storage of request bodies, response bodies, and streaming chunks. Both the client request and each channel execution are recorded. Request headers retain the existing sensitive-header masking. Streaming chunks are stored in AxonHub's event format, rather than as byte-for-byte network traffic.

Administrative and garbage-collection contexts continue to use the global policy. Other API keys keep their existing storage behavior. Payloads follow the configured storage backend and cleanup policy; this override does not extend retention. Existing requests made while body storage was disabled cannot be reconstructed.

This setting is separate from `storage_policy`, so saving the global policy through the UI does not erase the temporary capture. Invalid capture settings emit a warning and leave the global policy in effect. If capture stops early, check the service logs and the expiration. Confirm new target requests contain bodies and chunks, and confirm requests using another key still follow the global policy.
