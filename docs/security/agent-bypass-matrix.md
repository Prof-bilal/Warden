# Agent isolation bypass matrix

| Attack | Required result | Proof status |
| --- | --- | --- |
| direct `.env` read | blocked/invisible | Phase 0 |
| Python read | blocked/invisible | Phase 0 |
| Node read | blocked/invisible | Phase 0 |
| shell, grep, awk, sed, strings | blocked/invisible | Phase 0 |
| symlink to secret | blocked | Phase 0 |
| hard-link alias | blocked or explicitly rejected by workspace construction | Phase 0 |
| `../` traversal | blocked | Phase 0 |
| absolute path | blocked | Phase 0 |
| subprocess read | blocked | Phase 0 |
| `/proc/self/root` and `/proc/*/fd` | blocked | Phase 0 |
| inherited descriptor | closed or explicitly scoped | Phase 0 |
| archive/copy operation | source read remains blocked | Phase 0 |
| Git working-tree secret | blocked | Phase 0 |
| Git-history secret | blocked when `.git` is not granted; documented limitation otherwise | Phase 0 |
| secret environment variable | absent | Phase 0 |
| unauthorized network exfiltration | proxy/kernel blocked and audited | Phase 0 |
| allowed source read/write | succeeds | Phase 0 |
| allowed build/test toolchain | succeeds | Phase 0 |

The harness must assert both the operation result and the absence of secret bytes in
stdout, stderr, audit records, temporary outputs, and evidence artifacts.
