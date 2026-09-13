# Hacker News — Full Plan

**Account:** Use real name or consistent pseudonym with some karma (1–2 quality comments first)  
**Theme:** Technical, humble, architecture-focused — HN rewards honesty and punishes marketing  
**Link:** `github.com/Prof-bilal/Warden`

---

## 1. Before You Post

### HN Guidelines Checklist

- [ ] Read [HN Guidelines](https://news.ycombinator.com/newsguidelines.html) — especially self-promo (10:1 ratio)
- [ ] Build 10–20 karma with genuine comments before posting Show HN
- [ ] Don't ask friends to upvote — HN detects and penalizes
- [ ] Title must be factual, no superlatives, no clickbait
- [ ] Be ready to answer every question for 2–3 hours after posting

### Best Time to Post

| Time (IST) | Time (UTC) | Time (US Eastern) | Why |
|---|---|---|---|
| **12:30 PM IST** | 07:00 UTC | 03:00 AM ET | Early — catches EU morning + US early birds |
| **4:30 PM IST** | 11:00 UTC | 07:00 AM ET | **Best** — US East morning, EU afternoon, front-page window |
| 8:30 PM IST | 15:00 UTC | 11:00 AM ET | Good — US West morning |

**Recommended:** **4:30 PM IST (11:00 UTC / 7:00 AM ET)** on a Tuesday–Thursday.

---

## 2. Show HN Post — Day 4 (Sept 13, 4:30 PM IST)

### Submission

| Field | Content |
|---|---|
| **Title** | `Show HN: Warden – sandbox for MCP servers` |
| **URL** | `https://github.com/Prof-bilal/Warden` |
| **Text** | Leave empty (URL post, not text post) |

> Title must be ≤80 chars, no superlatives. `Show HN:` prefix is required for Show posts. Don't add "open source" or "free" to title — mention in comments.

### First Comment (Post Immediately — Within 1 Minute)

```
Warden is a single-binary CLI that runs MCP servers inside an OS-native sandbox. You write a policy granting the paths, hostnames, and env vars a server may use; everything else is denied at the OS layer.

Stack:
- Go
- Linux = bubblewrap (unshare user/ipc/pid/net, ro-bind /usr /lib64 + policy paths)
- macOS = sandbox-exec (Seatbelt)
- Windows = AppContainer token with WFP egress filters, Job-Object limits, ETW audit — fails closed if any layer can't initialize
- Docker --network none is the fallback
- Egress = local proxy that checks hostnames before DNS

Key properties:
- Deny by default (ungranted paths are invisible, not just permission-denied)
- Fail closed (refuses to run if no backend can be applied — the refusal is a tested path)
- Tested against 18 real MCP servers (14 pass, 2 conditional, 2 fail by design — we publish why)

Known limitations (all in docs/security.md):
- Proxy filters HTTP/HTTPS only; raw TCP/UDP relies on namespace/WFP
- No CPU limits
- macOS Seatbelt deprecated upstream
- Windows needs admin/elevation
- If unprivileged userns are disabled, network boundary can widen (documented gap)

Happy to answer questions about the architecture or tradeoffs.

Repo: https://github.com/Prof-bilal/Warden
Docs: https://prof-bilal.github.io/Warden/
```

**No image/video on HN** — it's text only. Link to GitHub; HN will use GitHub's OG image (set in repo settings).

---

## 3. Follow-Up Comments (Prepare in Advance)

### If Asked "Why not just use Docker?"

```
Docker is one of Warden's fallback backends. But for "run one npm script with a restricted home directory":

- Namespace sandboxing has a much smaller trust boundary (OS primitives vs container runtime + daemon)
- No daemon dependency — single static binary
- Near-zero startup overhead vs container cold start

On Linux/macOS/Windows, Warden prefers OS-native. Docker is the fallback when no native primitive exists. The policy file is the same either way.
```

### If Asked "How does the proxy work?"

```
It's an in-process HTTP proxy. The sandbox's network namespace has no default route (Linux/macOS) or WFP blocks everything but the proxy (Windows). All outbound traffic routes through the proxy, which checks the hostname against network.allow before any DNS lookup.

Denied hosts are never resolved — no DNS leak channel. The proxy filters HTTP/HTTPS; raw TCP/UDP blocking relies on no-route (Linux/macOS) or WFP (Windows).

Code: internal/proxy/proxy.go — hostname check is at the very top, before any dial.
```

### If Asked "What about unprivileged user namespaces?"

```
Good catch — if they're disabled, the Linux fallback may run with broader network access. This is documented in docs/security.md:60-65 — the doc says Warden "currently does not always detect it."

It's a known gap, not a silent failure. The right fix is detection + fail-closed, which is tracked in REMAINING_WORK.md.
```

### If Asked "Which servers don't work?"

```
From the compat matrix (18 servers):

- docker-mcp: inherent — granting the Docker socket voids the sandbox
- playwright: schema gap — no wildcard hosts, no unix-socket grants (browsers need many hosts)
- fetch, kubernetes: conditional — need per-deployment hosts

Full matrix with pinned policies: https://github.com/Prof-bilal/Warden/blob/main/docs/compatibility.md

Each has a policy.yaml fixture in testdata/compat/ — the test suite enforces they match the manifest.
```

### If Asked "Has it been audited?"

```
No third-party audit. Security model + known limitations are in docs/security.md, and the test suite includes escape tests (unlisted path invisible, denied host never resolved, fail-closed refusal).

But "has it been audited?" is the right question — the answer is no, and we don't claim otherwise.
```

---

## 4. Engagement Strategy

| Rule | Detail |
|---|---|
| Reply within 15 minutes | First 2 hours are critical for HN ranking |
| Be humble | Acknowledge limitations — HN rewards honesty, punishes hype |
| Link to source | Every claim → file:line or docs section |
| Don't be defensive | If someone finds a flaw, thank them |
| Don't ask for upvotes | HN penalizes vote manipulation |
| One submission only | If it doesn't make front page, don't repost for 2+ weeks |
| Monitor for 3 hours | Stay available for the full discussion |

### What to Do If It Hits Front Page

- [ ] Reply to every top-level comment within 30 minutes
- [ ] Thank commenters who ask good questions
- [ ] Correct misinformation politely with evidence
- [ ] Don't over-explain — short, factual answers
- [ ] If a bug is found, acknowledge and link to the issue

### What to Do If It Doesn't

- Don't repost immediately
- Wait 2+ weeks, then try again with a different angle (e.g., "How we sandbox MCP servers on 3 OSes")
- Share the HN link on X/Twitter: "Posted Warden on HN — would love feedback: [HN link]"

---

## 5. Image / Video Usage

| Asset | Used On HN? | Why |
|---|---|---|
| Images | **No** | HN is text only — no image uploads |
| Video | **No** | Link to YouTube in a comment if someone asks |
| GitHub OG image | **Yes (auto)** | Set in GitHub repo → Settings → Social preview → upload `warden-post-x-launch-1200x675.png` (1280×640) |

**GitHub Social Preview:** Upload `warden-post-x-launch-1200x675.png` (or `1280×640` variant) to GitHub repo Settings → Social preview. This is what HN shows as the thumbnail.

---

## 6. Posting Checklist

- [ ] Karma ≥10 (post 1–2 quality comments first)
- [ ] Title: `Show HN: Warden – sandbox for MCP servers` (exact)
- [ ] URL: `https://github.com/Prof-bilal/Warden`
- [ ] First comment ready to paste (copy from §2)
- [ ] Follow-up comments prepared (§3)
- [ ] Available for 3 hours after posting
- [ ] GitHub social preview image set
- [ ] No image/video to upload (text only)
