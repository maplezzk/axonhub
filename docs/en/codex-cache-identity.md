# Codex cache identity for compatible clients

The ChatGPT Codex backend uses the `Session-Id` header for Responses cache
affinity. Sending the same `prompt_cache_key` in the JSON body while changing
`Session-Id` on every request can prevent cache reuse.

AxonHub preserves explicit client identity in this order:

1. The incoming `Session_id` or `Session-Id` header.
2. `session_id` from `X-Codex-Turn-Metadata`.
3. A stable ID derived from an explicit, nonblank `prompt_cache_key` when the
   client does not provide the above identity.
4. The existing context session or a generated request ID when no cache key is
   available.

The derived ID is namespaced by AxonHub's trusted caller scope (API key and
project for API requests) and the upstream ChatGPT account. Different callers
using the same cache key therefore receive different derived IDs. The JSON
`prompt_cache_key` is preserved. Generated thread, conversation, window, and turn
metadata use the resolved session ID; explicit client metadata is preserved.

Clients should keep their cache key stable across turns of the same conversation
and choose another key for a different conversation. The fallback applies to
both Responses and compact requests and does not require body passthrough.
Matching identity improves cache affinity but does not guarantee a hit: prompt
prefixes, model, upstream account, cache availability, and load still matter.

## Validation

An HTTP transformation integration test covers repeated requests with different
gateway context IDs, caller and conversation isolation, explicit client header
precedence, blank-key fallback, and Responses/compact identity consistency.

A live Luna probe before the fix used eight identical 17,130-token requests on
one Codex channel. A fixed session gave cached-token counts
`0, 16128, 16128, 16128`; four new sessions gave `0, 0, 0, 0`. Raw upstream SSE
usage and gateway records agreed. Deployment verification should repeat requests
without any client session header, with one stable body cache key, to exercise
the new gateway fallback rather than manually supplying the desired header.
