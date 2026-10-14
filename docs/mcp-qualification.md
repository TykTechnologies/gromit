# MCP qualification policy

The opt-in `mcp-qualification` feature generates a release workflow caller for
the shared workflow in `TykTechnologies/github-actions`. It supports Gateway
(`tyk`), Dashboard (`tyk-analytics`), Pump (`tyk-pump`) and MDCB (`tyk-sink`)
repositories with an `api` test job. Other repositories and branches retain
their existing jobs and test selection.

Enable the feature in `config/config.yaml` only for branches whose selected
Dashboard revision includes the MCP runner and fixtures. Workflow revisions
are pinned in `mcp-qualification.gotmpl`.

Every managed release workflow subscribes to `ready_for_review`, independent
of the MCP flag. This lets existing draft-gated build and API jobs start when
a PR becomes ready.

When enabled, the policy:

- Calls the shared workflow at an immutable commit with `PROBE_APP_ID`,
  `PROBE_APP_PRIVATE_KEY` and `DASH_LICENSE` forwarded. MDCB callers also forward
  `MDCB_LICENSE`.
- Includes its result in the required aggregate job dependencies.
- Excludes MCP from the parallel release API suite only when the mandatory
  serial qualification job is generated.

The reusable workflow lives in `github-actions`; tests, requirements, runner
and compose fixtures remain under Dashboard's `tests/api`. The shared job
builds components from coordinated source revisions and runs the four
MessagePack/Protobuf × MongoDB/PostgreSQL combinations. Each caller exercises
its component's own PR head.

The generated caller, parallel API-suite exclusion and aggregate dependency
must remain together. Removing the qualification job while retaining the
exclusion leaves MCP coverage missing. Drift checks compare generated workflows
against the deployed Gromit policy.

## MDCB qualification

The sink caller selects `topology: mdcb`. Gateway, Dashboard and Pump callers
use the direct topology. Both topologies require Dashboard fixtures compatible
with the selected sources and a GitHub App with read access to the component
repositories. MDCB also requires sink access and its licence secret.

The MDCB profiles verify API, key and policy propagation through RPC, persisted
protocol metadata, exact signed 64-bit error boundaries and one completion
record per request. A data-plane hybrid Pump reads both Redis serializers and
sends raw records over RPC without a local persistence output. MDCB persists
the records with `forward_analytics_to_pump=false`.
