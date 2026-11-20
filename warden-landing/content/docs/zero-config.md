# Run the Filesystem MCP server without a policy file

Start on Linux with a current Node.js and npm installation and a Warden build that includes the reviewed Filesystem pack. Run `warden doctor --prove` to check the sandbox.

From a directory without a `warden.yaml`, start the server:

```bash
warden run npx @modelcontextprotocol/server-filesystem
```

Warden selects the catalog’s exact Filesystem version. On the first run it installs that release in a separate sandbox, with npm install scripts disabled and egress limited to the public npm registry. The running server has no network grant and speaks MCP over stdin and stdout.

By default, the server can read the data directory at `${XDG_DATA_HOME:-$HOME/.local/share}/warden/data/filesystem`. To select an existing directory, pass it as the final argument:

```bash
mkdir -p ./shared
warden run npx @modelcontextprotocol/server-filesystem ./shared
```

The data directory is read-only and Warden applies secret-file protection. The sandbox also reads the prepared runtime. For unattended jobs with a custom policy, save it as `warden.yaml` or select it with `--policy`. Use `warden run --dry-run` with that policy to inspect the effective grants.

When the bundled Filesystem version changes, Warden shows a policy diff and asks for approval before launching the new release.

For other launchers, see [the CLI reference](cli.md) for the policy lookup order and [the security model](security.md) for permissions that require review.
