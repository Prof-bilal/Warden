# Warden 2.0 launch drafts

Publish after the 2.0 tag, npm package, and linked pages are live.

## Show HN

**Title**

Show HN: Warden 2.0 – A local sandbox and install guard for MCP servers

**Post**

I built Warden to put a reviewable permission boundary around local MCP
servers. In 2.0, a reviewed Filesystem launcher can start under a bundled
read-only policy with one command. Other launchers can use an explicit policy.

The install guard checks a selected npm or PyPI release before installation.
It defaults to a 48-hour cooldown and uses a sandbox for installation. For npm,
only the reviewed top-level package’s exact version can be allowlisted for
install hooks. Warden runs those hooks with egress denied.

The MCP bridge also remembers the first definition of an allowed tool. If the
definition changes, the bridge hides that tool and blocks calls until you
review the change. The pin covers the published tool definition. A server can
still change its implementation without changing that definition.

I’d like feedback on which MCP workflows are hardest to run with narrow
permissions, and which install checks would be useful in CI.

[Source](https://github.com/Prof-bilal/Warden) · [Warden 2.0 walkthrough](https://warden.blog/v2)

## X thread

**1. Attach `warden-zero-config.mp4`**

Warden 2.0 is out. A reviewed Filesystem MCP server can run with one command and a read-only policy. The demo shows which reads and writes the sandbox permits. https://warden.blog/docs/zero-config

**2. Attach `warden-install-guard.mp4`**

Install Guard blocks releases inside its default 48-hour cooldown. This local registry fixture published a package two hours earlier. Warden stopped before installing it. https://warden.blog/docs/install-guard

**3. Attach `warden-mcp-tool-pins.mp4`**

A tool can keep its name and change its definition. Warden’s MCP bridge hides the changed tool, blocks direct calls and records the reason. Review the new hash before accepting it. https://warden.blog/docs/mcp-tool-pins

**4. Migration**

Upgrading an existing setup? `warden migrate` previews policy version and digest changes. It keeps backups when applying verified changes and flags launchers that need review. https://warden.blog/docs/migration

## Blog post

# Put an install check before your first MCP run

The first decision you make about an MCP server often happens before it starts.
A package manager can fetch dependencies and run installation hooks while
preparing the server. A runtime policy becomes useful only after that work has
already happened.

Warden 2.0 adds an install guard to that earlier step. In our local registry
fixture, a package released two hours earlier hit the default 48-hour cooldown.
The guard stopped before invoking the installer. The cooldown is configurable,
and a one-off override leaves an audit event.

### Review the exact release

Warden checks npm registry metadata and inspects the actual package archive.
An explicit `name@version` entry can authorize that package’s reviewed install
hooks. Warden starts installation with dependency scripts disabled. Approved
hooks run in a separate installation sandbox with egress denied.

For PyPI, the guard uses a reviewed, hash-checked binary wheel. Review dependent
wheels separately. Missing metadata can produce a warning or, in strict mode,
stop the install. These checks tell you about specific release and installation
properties. Runtime permissions still matter when the server runs.

### Keep reviewing after installation

Start the reviewed Filesystem server under the bundled policy, or provide your
own policy for a different launcher. The bridge can also pin tool definitions.
In a scripted two-session test, changing only the description hid the tool,
blocked a direct call and recorded the denial in the journal. Review the
definition hashes before accepting the change.

If your Warden policy lives in Git, the CI action can show which permissions a
pull request changes. Use `fail-on: widen` when those changes require review
before the job proceeds.

[Install Guard guide and CI recipe](https://warden.blog/docs/install-guard) · [Migration guide](https://warden.blog/docs/migration)
