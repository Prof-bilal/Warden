"use client";
import { useState } from "react";
import CodeBlock from "@/components/CodeBlock";
const decode = (s: string) =>
  Uint8Array.from(atob(s.replaceAll("-", "+").replaceAll("_", "/")), (c) =>
    c.charCodeAt(0),
  );
const digest = async (b: Uint8Array) =>
  Array.from(
    new Uint8Array(await crypto.subtle.digest("SHA-256", b as BufferSource)),
  )
    .map((v) => v.toString(16).padStart(2, "0"))
    .join("");
export default function EvidenceInspector() {
  const [text, setText] = useState("");
  const [key, setKey] = useState("");
  const [status, setStatus] = useState(
    "Choose evidence and an independently trusted public key.",
  );
  const [details, setDetails] = useState("");
  const [busy, setBusy] = useState(false);
  const check = async () => {
    setBusy(true);
    setDetails("");
    try {
      if (!/^[a-fA-F0-9]{64}$/.test(key.trim()))
        throw Error("Enter a 64-character public key from a trusted channel.");
      if (text.length > 1048576) throw Error("Evidence exceeds 1 MiB.");
      const envelope = JSON.parse(text);
      const payload = decode(envelope.payload),
        sig = decode(envelope.signature);
      const pub = Uint8Array.from(key.trim().match(/../g)!, (v) =>
        parseInt(v, 16),
      );
      const fingerprint = await digest(pub);
      if (envelope.key_id !== fingerprint)
        throw Error("Issuer differs from your trusted key.");
      const publicKey = await crypto.subtle.importKey(
        "raw",
        pub,
        { name: "Ed25519" },
        false,
        ["verify"],
      );
      if (!(await crypto.subtle.verify("Ed25519", publicKey, sig, payload)))
        throw Error("Signature is invalid.");
      const report = JSON.parse(new TextDecoder().decode(payload));
      if (report.version !== 1 || report.issuer !== fingerprint)
        throw Error("Invalid evidence schema.");
      const issued = Date.parse(report.issued_at),
        expires = Date.parse(report.expires_at);
      if (
        !Number.isFinite(issued) ||
        !Number.isFinite(expires) ||
        issued > Date.now() + 60000 ||
        expires <= Date.now() ||
        expires <= issued ||
        expires - issued > 14 * 86400000
      )
        throw Error("Evidence expired or timestamps are invalid.");
      setStatus(
        "Signature and expiry checked. Artifact bindings and revocation still need CLI verification.",
      );
      setDetails(
        JSON.stringify(
          {
            issuer: fingerprint,
            platform: report.platform,
            backend: report.backend,
            protocol: report.protocol,
            expires_at: report.expires_at,
            artifact_sha256: report.artifact_sha256,
            policy_sha256: report.policy_sha256,
            checks: report.checks,
          },
          null,
          2,
        ),
      );
    } catch (e) {
      setStatus(
        e instanceof Error
          ? e.message
          : "Verification unavailable in this browser.",
      );
    } finally {
      setBusy(false);
    }
  };
  return (
    <section className="max-w-4xl rounded-2xl border border-ink-700 p-6 sm:p-8">
      <label className="block text-sm">
        Signed evidence file
        <input
          type="file"
          accept="application/json,.json"
          className="mt-3 block w-full text-sm text-muted"
          onChange={async (e) => {
            const f = e.target.files?.[0];
            setDetails("");
            setStatus("Evidence loaded; enter your trusted issuer key.");
            if (f && f.size <= 1048576) setText(await f.text());
            else {
              setText("");
              setStatus("Choose a JSON file smaller than 1 MiB.");
            }
          }}
        />
      </label>
      <label className="mt-6 block text-sm">
        Independently trusted Ed25519 public key
        <input
          value={key}
          onChange={(e) => {
            setKey(e.target.value);
            setDetails("");
          }}
          autoComplete="off"
          spellCheck={false}
          className="mt-2 w-full rounded-lg border border-ink-700 bg-ink-950 p-3 font-mono text-sm"
          placeholder="64 hex characters from the creator's trusted channel"
        />
      </label>
      <p className="mt-3 text-xs leading-relaxed text-muted">
        A key supplied inside the evidence is not a trusted identity. Compare
        the fingerprint through a channel you already trust.
      </p>
      <button
        disabled={!text || busy}
        onClick={check}
        className="mt-6 rounded-lg bg-blueprint px-5 py-3 text-sm font-semibold text-ink-950 disabled:opacity-40"
      >
        {busy ? "Checking…" : "Inspect signature"}
      </button>
      <p role="status" className="mt-5 text-sm text-progress">
        {status}
      </p>
      {details && <CodeBlock className="mt-5">{details}</CodeBlock>}
      <p className="mt-6 text-sm leading-relaxed text-muted">
        This page checks the signature locally. It cannot inspect your installed
        server, trust an issuer for you, or check a private revocation feed. Run{" "}
        <span className="font-mono text-paper">warden creator verify</span> with
        your local artifacts and a fresh trusted snapshot before accepting a
        checks-passing badge. Nothing is uploaded or saved.
      </p>
    </section>
  );
}
