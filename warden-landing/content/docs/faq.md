# Troubleshooting and FAQ

Find your error message below. Each entry names the cause and the fix.

## `warden run` refuses to start

### "no sandbox backend available ... refusing to run unsandboxed"

Warden could not find a sandbox on this computer. Fix it for your system:

- **Linux:** install `bwrap` (bubblewrap). See [Install](install.md#per-os-prerequisites).
- **macOS:** check that `sandbox-exec` exists.
- **Anywhere else:** start the Docker daemon.

Warden never runs a program without a sandbox. This error is the protection working as designed.

### "command ... is not an absolute path"

The program has to be given as an absolute path. Bare names such as `npx`, `uvx`, or `node` are looked up on your `PATH`. If you see this error, the name is not on the `PATH` of the shell that starts Warden. Check with `command -v npx`, or put the full path in the policy's `command:` line.

### "executable ...: no such file or directory"

The path does not exist on this computer. This usually happens when a policy was copied from another machine, so its `command:` points at a different install. Change it to the location of your server file.

### "strace not found"

On Linux, Warden needs `strace` to audit programs. Install it with your package manager, for example `sudo apt install strace`. The [install page](install.md#per-os-prerequisites) lists the commands for each distribution.

## My program starts, but something is denied

### How do I see what was blocked?

```bash
warden logs --tail 20
```

Each line names the type (`file` or `network`), the action, the resource, and whether it was allowed, with the reason. A blocked path or host tells you exactly which grant to add.

### The policy looks right, but access still fails

- **The same path is in both `read` and `write`:** this is harmless. Check `warden logs` for the real denial. It is usually a parent folder that was never granted.
- **SQLite writes fail:** grant `write` on the folder that holds the database, not only the database file. Journal files sit next to it.
- **Child processes fail:** the sandbox covers the whole process tree, including child processes. The denial still shows up in the log with the exact resource.

### Can I approve access interactively instead of editing YAML?

Yes. Run `warden run --policy policy.yaml --approve -- <command>`. Network grants apply right away. Filesystem grants are saved to the policy file, and the program restarts, because folder access is set when it starts. Approval needs a terminal. Without one, Warden stops.

## Security questions

### Can an AI coding agent edit a project without seeing its secrets?

Yes. [`warden agent`](agent-workspace.md) gives the agent a temporary sanitized copy of the project. It hides the files and environment variables you declare as secret, and it asks you to review the changes before they reach your project. Use fake credentials while you test, and remember that only the values you declare are protected.

### I passed `HOME`, and now the program can read `~/.ssh`

That comes from the policy, not from Warden. Passing `HOME` together with a `filesystem.read` grant that covers `~/.ssh`, or any parent of it, exposes your keys. Grant only the exact credential files the program needs, such as `./config/.pgpass` and not `~`. Read the [security review](security.md#5-credential-exposure) before you ship a policy that passes `HOME`.

### `warden trace` recorded my secrets. Is that logged?

`trace` runs the program without the sandbox, so environment values can appear in the trace log. `warden init` writes only variable names into the policy, and the values stay in your shell. Treat raw trace logs as sensitive, and delete them after you generate the policy.

## Limits

### Wildcard hosts (`*.example.com`)

A single leading wildcard is supported: `network.allow` accepts `*.example.com` to allow a subdomain pattern. See the [policy schema](schema.md). Arbitrary hosts, such as browser-style servers that need any site, are out of scope. Daemon sockets such as `/var/run/docker.sock` have no grant type and are [not supported](compatibility.md#failures-classified).

### "Will this work with my server?"

Check the [compatibility matrix](compatibility.md) first. If your server is not listed, [file a compatibility report](beta.md#filing-a-compatibility-report). That is exactly the information the beta program collects.
