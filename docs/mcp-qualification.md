# MCP qualification policy

The opt-in `mcp-qualification` feature generates a small release workflow
caller for the public shared workflow in `TykTechnologies/github-actions`.
It is supported for Gateway (`tyk`), Dashboard (`tyk-analytics`) and Pump
(`tyk-pump`) with an `api` test job. Other repositories and branches retain
their existing jobs and test selection.

Production configuration enables the feature only for these three repositories'
`master` policies. Before merging or deploying this activation, merge
[Dashboard #6298](https://github.com/TykTechnologies/tyk-analytics/pull/6298)
so Dashboard `master` contains the runner and fixtures required by callers
without a matching Dashboard source branch. The shared workflow is pinned to
the authentication fix from
[github-actions #165](https://github.com/TykTechnologies/github-actions/pull/165).

Every managed release workflow subscribes to `ready_for_review`, independent
of the MCP flag. This lets existing draft-gated build and API jobs start when
a PR becomes ready.

When enabled, the policy:

- Calls the shared workflow at an immutable commit with only `PROBE_APP_ID`,
  `PROBE_APP_PRIVATE_KEY` and `DASH_LICENSE` forwarded.
- Includes its result in the required aggregate job dependencies.
- Excludes MCP from the parallel release API suite only when the mandatory
  serial qualification job is generated.

The reusable workflow lives in `github-actions`; tests, requirements, runner
and compose fixtures remain under Dashboard's `tests/api`. The shared job
builds all components from coordinated source revisions and runs the four
serializer/storage combinations. Keeping a caller in all three producer
repositories ensures the changed producer's own PR head is exercised.

Rollout order:

1. Publish the shared workflow commit pinned in `mcp-qualification.gotmpl`.
2. Ensure every selected Dashboard revision contains `run_mcp_v2_stack.py`,
   its helper tests and `fixtures/mcp_v2/compose.yml`. Review the required App
   read permissions and Dashboard license availability.
3. Add `mcp-qualification` to the intended repository/branch features in
   `config/config.yaml`. Do not enable older release branches lacking the
   Dashboard prerequisites.
4. Build and deploy gromit through the normal release process. Drift checks
   execute the deployed policy, rather than a PR's proposed template changes.
5. Regenerate the three consumer release workflows with that policy and remove
   the duplicated `.github/workflows/mcp-qualification.yml` in each consumer.
   Use the shared caller instead; keep its aggregate dependency.

For TT-18006, both `TT-18006-feature-branch` consumers and master-targeted PRs
need their actual drift-selected policy branch considered. A template support
PR alone cannot make a drift check using the currently deployed master policy
pass. No merge, deployment or repository protection change is implied here.

Rollback the caller, marker exclusion and aggregate dependency together.
Removing only the qualification dependency or job loses MCP coverage while
leaving the parallel API tests filtered.

## MDCB qualification (TT-18691)

Sink (`tyk-sink`) master additionally supports `mcp-qualification`. Its generated
caller selects `topology: mdcb`, forwards `MDCB_LICENSE`, and pins the candidate
shared workflow at `0a3d17d61706a47d1c91c3819380665711da7ffe`. Existing Gateway,
Dashboard and Pump callers keep their direct topology and previous immutable
workflow revision. Release-branch selection is unchanged.

The sink caller, `and not mcp` selection in its parallel API suite and
`aggregator-ci-test` dependency are generated together. PR #569 remains
independent; its branch and standalone exclusion policy are not modified.
If that policy merges before rollout, replace the standalone exclusion with
this qualification-backed policy when reconciling master.

Prerequisites are the sink Pump pin containing protocol analytics and signed
64-bit errors, the Dashboard MDCB runner and compatible fixtures, App read
access to sink, and Dashboard/MDCB licence secrets. The four MDCB profiles
cover MongoDB/PostgreSQL × MessagePack/Protobuf, with forwarding to Pump
disabled. Require API/key/policy propagation and persisted protocol metadata,
exact signed boundaries and one completion record per request.

Roll out sink/runner prerequisites, publish the shared-workflow commit, then
deploy gromit and sync the generated sink release workflow. Keep these PRs
linked; a drift check using deployed gromit cannot validate a proposed policy
until deployment. Roll back the generated caller, API-suite exclusion and
required aggregate dependency together, then roll back the policy pin.
Never remove the replacement qualification while retaining the exclusion.
