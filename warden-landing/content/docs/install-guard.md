# Review a package with Warden Install Guard

[Watch a two-hour-old fixture release get blocked](/videos/warden-install-guard.mp4).

Run a guard-enabled Warden build with npm and Node.js or with Python and pip.
Choose a package and a new destination path whose parent already exists.
Replace these example packages with the release you want to review.

```bash
warden guard npm install is-number@7.0.0 --output ./reviewed-npm --strict
warden guard pip install idna==3.10 --output ./reviewed-wheel --strict
```

The default cooldown is 48 hours. Set `minimum_release_age: 72h` in
`~/.config/warden/guard.yaml` to change it. Warden uses
`$XDG_CONFIG_HOME/warden/guard.yaml` when that variable is set.
An explicit `--override-age` applies to one invocation and is audited.
Allowlisted native builds require a local compiler, build tools and headers for
the selected Node.js installation.

| Check | Scope |
| --- | --- |
| Publication age | Blocks recent known releases. Missing age metadata warns by default and blocks with `--strict`. |
| npm install hooks and native build metadata | Checks the release metadata and archive, including `binding.gyp`. An exact `name@version` entry in `script_allowlist` authorizes that package’s reviewed hooks to run in a separate sandbox with egress denied. Warden installs dependencies with `--ignore-scripts`. |
| Maintainer changes and provenance | Warden warns about a maintainer change when the registry supplies both maintainer lists. Provenance is informational. |
| Install traffic | The package manager runs in a sandbox with registry egress. Use `warden logs --explain` to inspect proxy denials. The install does not assess what the package will do when run. |
| Python source builds and dependencies | Warden installs a reviewed, hash-checked binary wheel. Review each dependency separately. |

## Project policy

Add this to a version 2 policy and pass it with `--policy ./warden.yaml` when
running either install command.

```yaml
version: 2
guard:
  minimum_release_age: 72h
  strict: true
```

The policy can raise the user’s cooldown and require strict metadata checks.
Script allowlists come from user configuration. A PR that lowers the policy
cooldown or removes strict checks requires review when using `fail-on: widen`.

## Review policy changes in CI

On a Linux workflow running on `pull_request`, pass the Action a tracked policy
file and use `fail-on: widen`. Use a Warden release or workspace binary that
provides `warden validate` and the Markdown policy diff.

```yaml
permissions:
  contents: read
  pull-requests: write

jobs:
  sandbox:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - run: sudo apt-get update && sudo apt-get install -y strace bubblewrap
      - uses: ./.github/actions/warden-action
        with:
          policy: warden.yaml
          run: ./scripts/test.sh
          fail-on: widen
```

The example uses the local Action checkout. Set `warden-version` to a reviewed
release tag when downloading a published binary. The review appears in the job
summary and, with comment permission, on the pull request. A diff requiring
review fails the job with `fail-on: widen`.
