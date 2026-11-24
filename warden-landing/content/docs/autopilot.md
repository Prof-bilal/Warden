# Run an offline tool with Warden Autopilot

Autopilot is a Linux preview that generates a sandbox policy from an approved program and project directory. Use it to run a local task without writing YAML. The default workspace is read-only, and sessions run offline.

## Install and check the sandbox

Install the beta CLI on Linux. The preview was published in `0.2.0-beta.12`.

```bash
npm install -g warden-sandbox-cli@beta
warden --version
warden autopilot setup
```

Setup checks for `bwrap`, `setpriv`, and `strace`, then runs the native sandbox proof. If it reports a missing dependency or failed proof, resolve that before enrolling a real project. See the [Linux install requirements](install.md) and [verification notes](proof.md).

## Run a read-only task

Create a disposable project with one file and an npm marker. You can start in an existing project root once you’re familiar with the workflow.

```bash
cd "$(mktemp -d)"
printf 'before\n' > README.md
printf '{"name":"autopilot-demo"}\n' > package.json
warden autopilot plan
```

The plan reports recognized files and tools found on `PATH`. It can suggest commands based on the project markers. You choose what to run, and discovery leaves the project unchanged.

```bash
warden autopilot approve -- /usr/bin/cat
warden autopilot status
warden autopilot run -- README.md
```

Review the proposed policy and type `yes` in the terminal when prompted. `status` shows the saved enrollment, the generated policy, and the audit log location. `run` starts the approved `cat` executable and prints `before`. Run these commands from the same project root. Later launches reuse the approval, and everything after `run --` is passed to the enrolled program.

## Review an edit

Revoke the read-only enrollment before enrolling an editor with `--edit`. Try this in the disposable project from the previous step.

```bash
warden autopilot revoke
warden autopilot approve --edit -- /usr/bin/sed
warden autopilot run -- -i 's/^before$/after/' README.md
```

Type `yes` at enrollment. The program edits the temporary workspace. After it exits, Warden shows the proposed change. Type `apply` to copy the reviewed edit to the original `README.md`. Any other response discards it. Run `cat README.md` to check the result. See the [agent workspace guide](agent-workspace.md) for how Warden handles conflicts and protected data.

## Revoke access

```bash
warden autopilot revoke
```

Revocation blocks future launches for this project. Stop any active run separately. Use `warden logs --explain` to inspect the audit trail.

## Agent compatibility

Detection means Warden found an executable on `PATH`. On the tested Linux host, Claude Code and the direct Codex binary passed offline version checks. The Codex and OpenCode shell wrappers failed to start, and the direct OpenCode binary could not resolve a home directory. This preview supports offline tasks. Cloud-backed coding sessions were outside the benchmark.

For custom grants or another supported platform, use the [manual policy quickstart](quickstart.md). For optional advisor configuration, see the [Autopilot CLI guide](https://github.com/Prof-bilal/Warden/blob/v0.2.0-beta.12/warden-starter/warden/docs/autopilot.md).
