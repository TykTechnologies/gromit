# MCP qualification policy

The opt-in `mcp-qualification` feature generates a small release workflow
caller for the public shared workflow in `TykTechnologies/github-actions`.
It is supported for Gateway (`tyk`), Dashboard (`tyk-analytics`) and Pump
(`tyk-pump`) with an `api` test job. Other repositories and branches retain
their existing generated workflows.

When enabled, the policy:

- Calls the shared workflow at an immutable commit with only `PROBE_APP_ID`,
  `PROBE_APP_PRIVATE_KEY` and `DASH_LICENSE` forwarded.
- Includes its result in the required aggregate job dependencies.
- Excludes MCP from the parallel release API suite only when the mandatory
  serial qualification job is generated.
- Starts release CI when a draft becomes ready for review.

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
   Dashboard prerequisites. This support PR does not enable production policy.
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
