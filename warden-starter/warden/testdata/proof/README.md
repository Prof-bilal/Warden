# Warden proof harness (plan §6)

Reproducible evidence, not hand-written claims. The harness renders
`proof-policy.yaml.template` and the target template under `fixtures/`
into temp dirs, runs the target under `warden run`, and verifies the
deny-by-default contract gated on a positive control (the target must
provably start).

Run it (from the module root, `warden-starter/warden/`):

    go build ./cmd/warden
    bash testdata/proof/run-proof.sh ./warden

Artifacts land under `evidence/<platform>/<stamp>/` (for example
`evidence/linux/20260915-101500/`), containing `results.jsonl`,
run-scoped `audit.jsonl`, `summary.json`, and `evidence.md`. Set
`WARDEN_PROOF_OUT` to write the evidence tree somewhere else.

Isolation properties:

- The run's audit stream is captured via a private `XDG_STATE_HOME`
  scoped to the harness invocation, so the evidence is not mixed with
  other warden runs on the same machine.
- The env-leak canary (`WARDEN_SECRET_ENV`) is exported before the run;
  the `env_denied` step fails if it leaks into the sandbox.

The fixture target is harmless by construction: it touches only harness-created
temp files and loopback addresses (`127.0.0.1` HTTP server + a `.invalid` name
that cannot route anywhere).
