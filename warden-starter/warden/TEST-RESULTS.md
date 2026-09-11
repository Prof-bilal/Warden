# Warden Test Results

> **This hand-written results file is retired (audit P1-7).** Snapshot
> documents like the old "19/19 tests passed" version went stale as the code
> changed (e.g. the Linux strace requirement for `warden run`) and claimed
> more than could be reproduced.
>
> Evidence now comes from the **reproducible proof harness**:
>
>     go build ./cmd/warden
>     bash testdata/proof/run-proof.sh ./warden
>
> It runs a harmless fixture target under `warden run` and, gated on a
> positive control (the target must provably start), verifies the full
> deny-by-default contract: allowed reads/writes succeed, secret and unlisted
> files are blocked, allowlisted loopback HTTP succeeds, unlisted hosts get
> HTTP 403 from the egress proxy, allowlisted env vars are visible and
> unlisted ones absent. It writes `results.jsonl`, the run-scoped
> `audit.jsonl` stream, `summary.json`, and `evidence.md` under
> `evidence/<platform>/<stamp>/`.
>
> Unit and escape tests: see `TESTING.md` (`go test ./...`). CI additionally
> asserts on Linux/macOS/Windows that escape tests actually executed instead
> of silently skipping.
- Use Docker backend for consistent behavior
- Test security boundaries incrementally

## Next Steps

1. Test on macOS with Seatbelt backend
2. Test on Windows with AppContainer backend
3. Try real MCP servers (filesystem, github)
4. Create custom policies for production
5. Enable audit logging for production

## Conclusion

**Warden MCP sandbox is fully functional and secure.** ✅

- Multi-platform support working
- Security boundaries enforced
- MCP protocol supported
- Ready for production use

**Test Coverage:** 19 tests passed
**Security Status:** All boundaries verified
**Recommendation:** Ready for deployment
