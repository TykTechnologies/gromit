# MCP qualification policy

Master policies for Gateway, Dashboard, Pump and Sink use `mcp-api-tests`. The
API controller keeps the existing compatibility rows and adds four MCP module
shards per MessagePack/Protobuf × MongoDB/PostgreSQL profile. All rows execute
inside the existing `api-tests` job through the standard environment checkout,
test checkout, `env-up`, API execution and reporting actions. Sink uses MDCB;
the other callers use the direct topology.

Component PR builds publish both their PR-head SHA tag and matching OCI revision
label. The controller resolves coordinated branch heads once, waits for those
images, then freezes registry digests and the `tyk-pro` environment revision.
It uses the exact caller PR head, matching peer branches and the existing
base-branch fallback. Unavailable images fail rather than using mutable tags.
The mock source is immutable; its image is published once when absent.

MCP rows use the versioned `tyk-pro/mcp.yml` Docker profile and the existing
datastore/cache overlays. Linux host networking preserves localhost upstream
fixtures; each row owns its runner, stores, mock, generated configurations and
RPC identity. The Dashboard-owned driver uses the same fixture data and
qualification checks as the native runner, without building applications in
test jobs. Native single-shard entrypoints remain supported for local use.

Regular API compatibility rows exclude MCP because the same API matrix supplies
the complete MCP shard rows. Tests remain serial within each MCP stack.
Reporting retains sanitized inventories and JUnit with shard-specific names,
removes the RPC identity and stops only that row's Compose project.

`aggregator-ci-test` depends on `api-tests` and its controller. Before aggregating
job statuses it validates all 16 MCP reports against frozen source/image and
environment revisions, full collection, completed IDs, declared skips and
routing/persistence checks. No separate `mcp-qualification` job is generated.
Non-PR release behavior and nightly test selection retain their existing scope.

The older `mcp-qualification` feature and shared workflow remain compatible for
branches that still use them. They are not enabled alongside `mcp-api-tests`.

Roll out Dashboard driver support and the `tyk-pro` profile first, publish the
shared action revision, then deploy this policy and regenerate callers. Every
new API action interface uses the same immutable shared-action commit. Rollback
restores the previous generated workflows and policy flags together.
