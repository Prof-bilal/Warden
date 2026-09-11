# Warden proof harness (plan §6)

Reproducible evidence, not hand-written claims. The harness renders
`proof-target.sh.template` and `proof-policy.yaml.template` into temp dirs,
runs the target under `warden run`, and verifies the deny-by-default contract
gated on a positive control (the target must provably start).

Run it:

    go build ./cmd/warden
    bash testdata/proof/run-proof.sh ./warden

Artifacts land under `evidence/<platform>/<stamp>/`:
`results.jsonl`, run-scoped `audit.jsonl`, `summary.json`, `evidence.md`.

The fixture target is harmless by construction: it touches only harness-created
temp files and loopback addresses (`127.0.0.1` HTTP server + a `.invalid` name
that cannot route anywhere).