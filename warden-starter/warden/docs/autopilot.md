# Warden Autopilot

Autopilot enrolls an installed agent or local development program and launches it in a Warden workspace projection. It uses Warden's per-session supervisor. The preview is designed for offline project tasks.

## Set up and enroll on Linux

Install Warden using the [existing installation guide](install.md), or build the CLI from this checkout:

```sh
cd warden-starter/warden
go build -o /tmp/warden-autopilot-preview ./cmd/warden
/tmp/warden-autopilot-preview autopilot setup
```

Use `warden` below if it is on your PATH, or substitute the path to your built binary. Setup checks for bubblewrap, setpriv and strace and runs the existing native containment proof against disposable fixtures. System dependencies and host security settings remain under operator control. Resolve a failed proof before using Autopilot for project work.

```sh
cd /path/to/project
warden autopilot plan
warden autopilot approve -- /usr/bin/cat
warden autopilot status
warden autopilot run -- README.md
warden logs --explain
```

Use a trusted installed executable of your choice in place of `/usr/bin/cat`. The example reads a project file called `README.md`. Run the commands from the same project root each time. Setup uses existing Warden auditing. `plan` reads fixed root-level marker metadata and checks a list of tools on PATH. Example build and test commands are hints. Inspect the project to confirm its actual tasks and dependencies. Approval requires typing `yes` on the controlling terminal. Later runs use the enrolled executable, with per-run arguments following `--`.

Add `--edit` when approving to make the projection writable. After a successful session, Warden displays eligible changes and prompts before applying them to the original. A rejected review or lack of a controlling terminal discards the candidate. See the [workspace guide](agent-workspace.md) for file type, size, apply and conflict behavior.

```sh
warden autopilot revoke
```

Revocation removes the saved authorization for future launches. Stop an active run separately. Use the original installation method to remove Warden itself.

## Try a native development build

The macOS and Windows paths in this checkout are under native verification. Build a local binary to try them with a disposable project. Keep the checkout and Warden's state outside the enrolled executable's directory. On macOS, keep the project and state outside the host temporary directories.

**macOS (Seatbelt)**

```sh
cd warden-starter/warden
go build -o "$HOME/warden-dev" ./cmd/warden
"$HOME/warden-dev" autopilot setup
cd /path/to/disposable/project
"$HOME/warden-dev" autopilot plan
"$HOME/warden-dev" autopilot approve -- /bin/cat
"$HOME/warden-dev" autopilot run -- README.md
```

**Windows (AppContainer)**

In PowerShell, run the commands from the repository root. This example uses a trusted `node.exe` installed in a directory you own as an offline file reader.

```powershell
Set-Location warden-starter/warden
go build -o "$env:LOCALAPPDATA\warden-dev.exe" ./cmd/warden
$warden = "$env:LOCALAPPDATA\warden-dev.exe"
& $warden autopilot setup
Set-Location C:\path\to\disposable\project
& $warden autopilot plan
& $warden autopilot approve -- (Get-Command node.exe).Source
& $warden autopilot run -- -e "process.stdout.write(require('node:fs').readFileSync('README.md'))"
```

Both examples expect a `README.md` in the disposable project and require confirming enrollment in the terminal. Run `warden autopilot revoke` from the project root when finished. Windows enrollment accepts installed `.exe` programs. Run the native setup proof before enrolling and resolve any containment error before continuing.

## Optional AI advice

Export `WARDEN_AUTOPILOT_ENDPOINT` as a complete OpenAI-compatible chat-completions URL and `WARDEN_AUTOPILOT_MODEL` as the installed or remote model name. Where required, provide `WARDEN_AUTOPILOT_API_KEY` through your host credential environment. HTTPS and HTTP on a loopback IP are supported. For local Ollama, an example endpoint is `http://127.0.0.1:11434/v1/chat/completions`. [Ollama compatibility](https://docs.ollama.com/api/openai-compatibility)

```sh
warden autopilot approve --advisor -- /usr/bin/cat
```

The host-side request carries the selected model and recognized marker names. The advisor can suggest read-only or reviewed edits. Provider failure or an invalid reply selects the read-only proposal. The same terminal confirmation applies. Provider settings and keys are used during advice and kept out of enrollment and the agent environment.

## Current limits

Autopilot has a fixed offline policy in this preview. Cloud sessions of Claude Code, Codex and OpenCode typically require inference access, credentials, and client-specific dependencies. End-to-end support for those workflows depends on a reviewed inference broker and native client tests. Offline version checks on Linux passed for Claude Code and the direct Codex binary. The tested Codex and OpenCode shell wrappers failed inside the sandbox, and direct OpenCode startup failed when it could not resolve a home directory. The proposal path detects the installed tool names. It does not rewrite their settings. Warden's [MCP connection workflow](mcp-gateway.md) handles separately configured tool authorization. Autopilot excludes selected project-root client configs, and an offline program may still start local tools within its OS sandbox.

[Warden security policy](https://github.com/Prof-bilal/Warden/blob/main/SECURITY.md)
