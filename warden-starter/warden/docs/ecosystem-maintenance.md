# Upgrades, reports and distribution

Use the development CLI until a reviewed release includes these commands. Existing
pack and client catalogs remain searchable; eight named adapters plus generic JSON
are available. Cline extensions and legacy Cascade need an explicit active file.
Config fixtures do not certify every application version or platform.

## Review an upgrade

```bash
warden policy-diff --before reviewed.yaml --after candidate.yaml
warden inventory --client cursor --scope project
```

Diff reports added/removed read/write paths, network hosts and environment names,
plus command changes and expanded resource limits. Any new grant, command change,
limit expansion or change in legacy MCP authorization requires review. It never
prints command arguments or credential values. The diff is conservative, not a
semantic filesystem-equivalence proof. Review gateway rules separately too.

Managed launchers pin the policy digest; changes require undo/review/re-wrap.
Inventory is read-only and starts no servers. It detects policy/launcher drift
for configured stdio entries; it does not discover all inherited plugins/direct
HTTP bypasses. Endpoint monitoring, automatic dependency updates and full host
version detection remain release work. Creator evidence refuses changed artifacts.

For rollback keep the previous prepared artifact, reviewed policy and private
configuration backup. `unwrap` restores the original launcher while preserving
unrelated edits; conflict detection stops overwrite of a changed managed launcher.
Re-run positive and denial tasks after upgrades. Never retry unsandboxed to fix a
connection failure. Keep ownership, review date, support scope and revocation
contacts with each pack contribution.

## Local pilot events

```bash
warden report record --file pilot.jsonl --client cursor --event setup-completed --seconds 90
warden report record --file pilot.jsonl --client cursor --event first-protected-task
warden report summary --file pilot.jsonl
```

Allowed events: setup-completed, setup-failed, first-protected-task, repeat-use,
rollback and badge-activation. These commands write owner-private local records
with coarse hour timestamps, a known client ID, an enum event and duration. No
prompt, tool body, credential, endpoint, account/user identifier or automatic
network telemetry is collected. Counts represent explicitly recorded events,
not unique users or certified activations. The [maintenance page](/maintenance)
can summarize a local report in-browser without uploading it.

## Incident and shutdown

The gateway journal records time, a random connection correlation ID, a normalized
method, allow/deny and a fixed reason. It excludes payloads, tool names, URIs and
tokens. The existing backend audit log is separate and can contain filesystem
paths/hosts; review it locally before sharing. Keep journals outside server grants.

```bash
warden stop --control-file /private/control.json
```

Control credentials are private and unique to the running gateway. Stop closes
managed sessions/processes. It cannot cancel unrelated direct connections or
undo already completed provider effects. Do not expose the control route through
your public reverse proxy. Restart rotates the control secret; old records fail.

## Contribution and release process

Use the policy-pack and compatibility issue templates. Report versions, backend,
fixture/workflow distinction, sanitized error category and reproducible fake-data
steps. Never paste configuration backups, env values, tool payloads or access tokens.
The creator CI template uses an ephemeral test key, not a production issuer.

Each release needs named pack/adapter maintainers, fresh allowed/denied evidence,
update/revocation review, rollback instructions, native platform checks and real
host-version coverage. A live pilot and its activation targets require actual users;
this checkout does not fabricate pilot outcomes. MCP bundles/plugins require a
host-specific packaging review and signing/distribution process. No universal
bundle or automatic publication is implied by the command/HTTP routes.
