# Running Linux Sandbox Tests Locally

This guide walks you through running the Linux bubblewrap escape tests on your local machine. These tests **must actually execute** (not skip) to verify the sandbox security boundary.

## Why Local Testing?

GitHub Actions' `ubuntu-latest` runner doesn't support unprivileged user namespaces by default, causing all Linux sandbox tests to skip. Running them locally on real hardware is the fastest way to verify the Linux backend works correctly.

## Prerequisites

- Linux distribution (Ubuntu 20.04+ recommended)
- `sudo` access (required to install packages and enable user namespaces)
- Go 1.26+
- ~5-10 minutes

## Installation & Setup

### 1. Install System Dependencies

```bash
sudo apt update
sudo apt install -y bubblewrap strace apparmor-profiles
```

Verify installation:
```bash
command -v bwrap
command -v strace
# Both should output paths to the binaries
```

### 2. Enable User Namespaces (if needed)

Some systems restrict unprivileged user namespaces. Check if they're enabled:

```bash
cat /proc/sys/kernel/unprivileged_userns_clone
# 0 = restricted (you may need to enable)
# 1 = enabled (you're good)
```

If restricted, enable them:
```bash
sudo sysctl -w kernel.unprivileged_userns_clone=1
# To make permanent:
sudo echo "kernel.unprivileged_userns_clone=1" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

### 3. Install Go 1.26+

If you don't have Go installed, download and install it:

```bash
wget https://go.dev/dl/go1.26.8.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.26.8.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

Verify:
```bash
go version
# Should output: go version go1.26.8 linux/amd64 (or higher)
```

### 4. Clone & Navigate to Warden

```bash
git clone https://github.com/Prof-bilal/Warden.git
cd Warden/warden-starter/warden
```

## Running the Tests

### Run All Linux Sandbox Tests

```bash
go test -v -count=1 ./internal/sandbox/linux/...
```

Expected output (all should **PASS**, never **SKIP**):
```
=== RUN   TestSandboxPositiveControlStartup
--- PASS: TestSandboxPositiveControlStartup (0.XX s)
=== RUN   TestSandboxReadGrantAccessible
--- PASS: TestSandboxReadGrantAccessible (0.XX s)
=== RUN   TestSandboxUnlistedPathInvisible
--- PASS: TestSandboxUnlistedPathInvisible (0.XX s)
=== RUN   TestSandboxWriteGrantWritable
--- PASS: TestSandboxWriteGrantWritable (0.XX s)
=== RUN   TestSandboxWriteOutsideGrantDenied
--- PASS: TestSandboxWriteOutsideGrantDenied (0.XX s)
=== RUN   TestSandboxExitCodePropagated
--- PASS: TestSandboxExitCodePropagated (0.XX s)
=== RUN   TestSandboxEnvPassthrough
--- PASS: TestSandboxEnvPassthrough (0.XX s)
=== RUN   TestRunFailsLoudWithoutBwrap
--- PASS: TestRunFailsLoudWithoutBwrap (0.XX s)
ok  	github.com/warden-sandbox/warden/internal/sandbox/linux	1.234s
```

### Run a Specific Test

```bash
go test -v -count=1 -run TestSandboxPositiveControlStartup ./internal/sandbox/linux/...
```

### Run Full Test Suite (including other packages)

```bash
go test -count=1 ./...
```

## Troubleshooting

### Tests are still skipping with "bwrap sandbox unusable"

```bash
bwrap --unshare-user --unshare-net --uid 0 --gid 0 \
    --ro-bind /usr /usr --ro-bind /lib /lib --ro-bind /lib64 /lib64 \
    --ro-bind /bin /bin /bin/true
```

If this fails, check:
1. User namespaces are enabled: `cat /proc/sys/kernel/unprivileged_userns_clone`
2. AppArmor isn't blocking bwrap: `sudo aa-status | grep bwrap`
3. Your kernel supports namespaces (most recent kernels do)

### "Permission denied" or AppArmor errors

Try temporarily disabling AppArmor for testing:
```bash
sudo systemctl stop apparmor
# Run tests
sudo systemctl start apparmor
```

Or use the AppArmor profile from the CI workflow (see `.github/workflows/ci.yml` lines 54-78).

### Tests fail instead of skip

If you see actual test failures (not skips), there's a real sandbox bug—file an issue with the output.

## Next Steps

Once local tests pass:

1. **Document the result** — Add a comment to your PR/commit confirming you've locally verified Linux tests pass
2. **(Optional) Set up GitHub Actions with a self-hosted runner** — Add your machine as a self-hosted runner so CI also validates Linux:
   - Go to: `https://github.com/Prof-bilal/Warden/settings/actions/runners/new`
   - Follow the "Add a new self-hosted runner" steps
   - Update `.github/workflows/ci.yml` line 23: `runs-on: self-hosted` (instead of `ubuntu-latest`)

## References

- [Linux Sandbox Implementation](./internal/sandbox/linux/)
- [CI Workflow](../.github/workflows/ci.yml)
- [Bubblewrap Documentation](https://github.com/containers/bubblewrap)
- [Linux User Namespaces](https://man7.org/linux/man-pages/man7/user_namespaces.7.html)
