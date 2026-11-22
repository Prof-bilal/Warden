# Review changes to MCP tool definitions

[Watch the bridge hide a changed tool](/videos/warden-mcp-tool-pins.mp4).

Enable persistent pins on a Warden bridge by giving its upstream a unique name:

```bash
warden connect --pin-server project-files --rules ./rules.json --policy ./warden.yaml
```

The policy must supply the local upstream command. You can also give
`warden connect` a prepared command after `--`. Pinning is available on
`warden connect` and `warden serve`. Each allowed tool’s first observed
definition becomes its baseline. Review the upstream before creating that
baseline.

If a pinned definition changes, Warden hides the tool from discovery, denies
its calls and records the change in the bridge journal. Inspect the hashes
before approving the new definition at a terminal:

```bash
warden mcp pin --server project-files --diff read_file
warden mcp pin --server project-files --accept read_file
```

A hash embedded in the rules file takes precedence and must be updated through
your rules review. Use the bridge for this feature. The legacy `warden proxy`
and the transparent `warden run` launcher do not track tool definitions.
