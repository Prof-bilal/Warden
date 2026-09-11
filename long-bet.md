# Long Bet — AI Agent & Browser Isolation for Full-Computer-Access Agents

> **Status:** Research-grounded strategy sketch, 2026-09-10. Synthesizes Warden's actual shipped surface with independent web verification of the full-computer-access AI agent landscape, the security risks these agents introduce, and how Warden could solve the isolation problem that currently forces users to buy VPS or configure virtual machines. Where a claim has no source it is marked **Open question**. This doc is **not** a commitment to build — it is a sequenced bet with evidence.

**How to use:** One long-term direction for Warden that is explicitly **not** a near-term roadmap item. Tagged `Speculative` (bet needing signal). Sections cite the file + line for the Warden mechanism they would reuse.

---

## 0 — The Problem

AI agents that control a desktop (see [Anthropic’s computer use](https://www.anthropic.com/news/3-5-models-and-computer-use)) can see screens, move mice, click buttons, and type — full desktop control. Running these agents on your main computer is dangerous: a malicious instruction hidden in a website could steal passwords, send emails, or exfiltrate files. Currently, users must either buy a VPS ($5–50/mo) or configure a VM (complex setup, resource-heavy). Neither is simple.

---

## 1 — The Agents

- **Claude Computer Use** (Anthropic): Full desktop control via screenshots + mouse/keyboard.
- **OpenAI Operator**: Browser-only agent with computer-use capabilities.
- **AutoGPT**: Autonomous agent that breaks goals into subtasks using web/browser tools.
- **OpenClaw**: General-purpose agent platform.
- **Odysseus** (Pewdiepie): Unverified agent mentioned in content.

All require full computer access to operate.

---

## 2 — Security Risks

- **Prompt injection**: Attacker-hidden instructions in web content can be executed (OpenAI states this is “unfixable”).
- **Same-origin bypass**: Agentic browsers remove browser security protections.
- **Credential theft**: Agents with file access can exfiltrate cookies/passwords.
- **Data exfiltration**: File system access enables sensitive data leaks.
- **Supply chain**: Vulnerable browser automation libraries (CVE-2025-47241 affected 1,500+ projects).

---

## 3 — Current Solutions

- **VPS**: Rent a cloud computer (AWS EC2, DigitalOcean). Pros: complete isolation. Cons: cost, technical barrier.
- **Local VM**: Run VirtualBox/QEMU locally. Pros: free, complete isolation. Cons: complex setup, resource-heavy.
- **Docker**: Container-based. Pros: lightweight. Cons: no GUI support, not ideal for full desktop control.

None are simple, purpose-built for AI agents, nor auditable.

---

## 4 — The Warden Opportunity

A new `warden agent` command would run any AI agent (Claude Computer Use, AutoGPT, etc.) in an isolated environment defined by the same `policy.yaml` model used for MCP servers.

```bash
warden agent run --policy policy.yaml -- agent-command
```

---

## 5 — Architecture

- **Tiers**: Tier 1 (namespace/bwrap), Tier 2 (container/Docker), Tier 3 (microVM/Firecracker).
- **Display**: Virtual display (Xvfb) for screen access.
- **Policy**: Same `policy.yaml` restricts filesystem, network, env, and limits.
- **Audit**: Log every agent action.
- **Approval**: Human-in-the-loop for sensitive actions.

---

## 6 — Competition

- **AWS EC2**, **DigitalOcean**: Cloud VMs (fully isolated but costly).
- **VirtualBox/Docker**: Local VMs/containers (free but complex).
- **Daytona**: Agent sandbox (microVM, paid).
- **Warden** (current): MCP sandbox (namespaces only).

Warden differentiates with the same policy model and local-first approach.

---

## 7 — Sequencing

- **Phase 0**: This document (now).
- **Phase 1**: Tier 1 agent mode (namespace + virtual display).
- **Phase 2**: Tier 2 (container isolation).
- **Phase 3**: Tier 3 (microVM) + approval flow.
- **Phase 4**: Marketplace / policy templates.

---

## 8 — Sources

- Anthropic: [computer use announcement](https://www.anthropic.com/news/3-5-models-and-computer-use)
- OpenAI Operator: [operator launch](https://openai.com/index/introducing-operator)
- AutoGPT: [Wikipedia](https://en.wikipedia.org/wiki/AutoGPT)
- Techtimes: [AI browser security review](https://techtimes.com/articles/318528/20260616/ai-browser-comparison-2026)
- Axis Intelligence: [browser agent security guide](https://axis-intelligence.com/browser-agent-security-risk-guide/)
- PiunikaWeb: [BioShocking credential leak](https://piunikaweb.com/2026/06/25/chatgpt-atlas-perplexity-comet-ai-browsers-leaking-credentials/)
- OWASP: [LLM Top 10](https://owasp.org/www-project-top-10-for-large-language-model-applications)
- NIST AI RMF: [nist.gov/itl/ai-risk-management-framework](https://www.nist.gov/itl/ai-risk-management-framework)
- Amazon v. Perplexity: [Ninth Circuit case](https://artificialintelligenceact.eu/)
- Daytona: [microVM sandbox](https://daytona.io/)
- Firecracker: [firecracker-microvm.github.io](https://firecracker-microvm.github.io/)
- Xvfb: [x.org docs](https://www.x.org/releases/X11R7.7/doc/man/Xvfb)

---

*Long Bet: Build Warden agent isolation for full-computer-access AI agents.*