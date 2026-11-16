# Quickstart: run a program in a sandbox

This takes about five minutes. You will write a small policy, run a program inside Warden, and see a blocked access in the audit log.

**Before you start:** install Warden and make sure `warden doctor` says **READY** (see [Install](install.md)).

You do not need to know YAML or sandboxing first. Each step says what it does and what you should see.

## 1. Make a test folder

```bash
mkdir -p warden-test/data && cd warden-test
echo "hello from data" > data/notes.txt
echo "top secret" > secret.txt
```

You now have one file the sandbox will be allowed to read (`data/notes.txt`) and one it will not (`secret.txt`).

## 2. Write a policy

Save this as `policy.yaml`:

```yaml
command: ["/bin/cat", "./data/notes.txt"]
filesystem:
  read: ["./data"]
  write: []
network:
  allow: []
env:
  allow: []
```

What it says:

- `command`: the program to run. Here it is `cat`, which prints a file.
- `filesystem.read`: the only folder the program can see.
- `network.allow` and `env.allow`: empty, so the program gets no network access and no environment variables.

Relative paths are resolved from the folder that contains the policy file.

## 3. Run it in the sandbox

```bash
warden run --policy ./policy.yaml
```

Expected output:

```
hello from data
```

The file was readable because it is inside the granted folder.

## 4. See what the sandbox blocks

Now try to read the secret file. Change the `command` line in `policy.yaml` to:

```yaml
command: ["/bin/cat", "./secret.txt"]
```

Run it again:

```bash
warden run --policy ./policy.yaml
```

Expected output:

```
cat: ./secret.txt: No such file or directory
```

The file is not hidden behind a permission error. It does not exist inside the sandbox, because it is not in a granted folder. Change the command back to `./data/notes.txt` when you are done.

## 5. Check the audit log

```bash
warden logs --tail 5
```

Each line is one access the program attempted, in JSON, with `"allowed":true` or `"allowed":false`. Blocked lines name the exact path or host, so when a program needs something new, you add that one grant to the policy and run again.

## Next steps

- **Your own MCP server:** set `command` to how you normally start it, for example `["/usr/bin/node", "/path/to/server/index.js"]`. Use an absolute path to the server file. Add only the folders, hosts, and environment variables it needs. [Example policies](examples.md) has copy-paste starting points for common servers, and the [compatibility matrix](compatibility.md) lists tested servers.
- **Not sure what a server needs?** `warden trace -- <command>` runs the program once without the sandbox and records what it touches. `warden init` then writes a draft policy from that recording. Treat the draft as a starting point. It may be missing grants, so review it and run `warden run` again to check.
- **Coding agents:** see [Protect an AI agent](agent-workspace.md).
- **Something failed:** see the [FAQ](faq.md).
- **Field-by-field reference:** [Schema](schema.md). **All commands:** [CLI reference](cli.md).
