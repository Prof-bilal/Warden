# Reddit — Full Plan

**Account:** Use existing account with karma, or new account (build karma first)  
**Theme:** Honest, technical, show-don't-shill  
**Link:** `github.com/Prof-bilal/Warden`

---

## 1. Before You Post

### Subreddit Rules Checklist

| Subreddit | Read Rules | Self-Promo Rule | Best Day/Time (IST) |
|---|---|---|---|
| r/MCP | [ ] | Must be relevant, no spam | Tue–Thu, 10 AM–2 PM |
| r/netsec | [ ] | Tool posts OK if technical | Tue–Thu, 10 AM–2 PM |
| r/commandline | [ ] | Show posts welcome | Any day, 10 AM–4 PM |
| r/LocalLLaMA | [ ] | Must tie to local AI | Mon–Fri, 9 AM–5 PM |
| r/programming | [ ] | No blogspam, must be technical | Tue–Thu, 10 AM–2 PM |
| r/golang | [ ] | Must be Go-related | Any day |

### Karma Building (If New Account)

- Comment on 5–10 posts in target subreddits (genuine, helpful)
- Wait 3–7 days before posting self-promo
- Reddit's 10:1 rule: 10 non-promo comments per 1 promo post

---

## 2. Main Technical Post — Day 3 (Sept 12, 10:00 AM IST)

> Post to **one subreddit at a time** (not all at once — looks spammy). Start with r/MCP, then r/netsec next day.

### Title Options (Pick One Per Subreddit)

| Subreddit | Title |
|---|---|
| r/MCP | `[MCP Security] I built a sandbox runtime for MCP servers — OS-level enforcement, deny by default, open source` |
| r/netsec | `[Tool] Warden — sandbox runtime for MCP servers using bubblewrap/Seatbelt/AppContainer` |
| r/commandline | `[Show] Warden — sandbox MCP servers with a YAML policy and OS-level enforcement` |
| r/LocalLLaMA | `Running MCP servers locally? They have full access to your system. I built a sandbox to fix that.` |
| r/programming | `I sandboxed every MCP server I run — here's the tool I'm building (open source, pre-1.0, Go)` |

### Post Body (Markdown — Use Verbatim)

```
I got tired of the gap between "MCP is great" and "an MCP server is just a process with my user's permissions". The protocol doesn't isolate anything, so I've been building Warden — a sandbox runtime for MCP servers.

**How it actually works (no marketing):**

You write a `policy.yaml` listing `filesystem.read`, `filesystem.write`, `network.allow` (bare hostnames only, no wildcards), `env.allow`, and `limits`. Warden translates that into OS primitives: bubblewrap with unshared namespaces on Linux, `sandbox-exec` on macOS, and on Windows an AppContainer token + WFP egress filters + Job Object + ETW audit. Docker `--network none --read-only` is the fallback.

**Details I think are worth discussing:**

- Denied hostnames are checked against the allowlist before DNS, so they're never resolved at all (no DNS leak channel).
- Ungranted filesystem paths are invisible, not "permission denied".
- Empty `env.allow` = empty environment. Nothing inherited.
- If no backend can be applied, it refuses to run. The refusal message is in a tested path.

**Honest caveats (docs/security.md lists these):**

- The egress proxy only filters HTTP/HTTPS
- No CPU limits
- If unprivileged userns are disabled the network boundary can widen
- macOS Seatbelt is deprecated upstream
- Windows needs an elevated shell

I also published the failures: the compat matrix covers 18 servers — 14 pass, fetch/kubernetes are conditional, docker-mcp and playwright are not sandboxable without schema changes (docker needs the daemon socket, which voids the sandbox).

I'd like honest breakage reports more than stars: `warden trace -- <your server>` then `warden run --policy policy.yaml`. If something doesn't fit the schema, that's exactly the feedback the project is collecting.

**Repo:** github.com/Prof-bilal/Warden
**Docs:** prof-bilal.github.io/Warden/
```

**Image (Optional — Attach One):**

| Image | Dimensions | Source |
|---|---|---|
| `warden-post-reddit-1200x630.png` | 1200×630 | Dark `#10141A` card, blue shield, white headline `How Warden sandboxes an MCP server`, policy code snippet faint bg. Generate via Figma or screenshot PH gallery `03-policy.html`. |

Alt text not needed on Reddit (no alt field), but keep image simple — Reddit prefers text posts.

**Flair:** Use subreddit flair if available: `Showcase`, `Tool`, `Project`

---

## 3. Discussion Question — For Engagement (Post as Comment or Separate Post, Day 10)

**Title (if separate post):**
```
MCP servers run with your full user permissions today. What would you grant a server you just installed from npm?
```

**Body:**
```
MCP servers run with your full user permissions today. What would you grant a server you just installed from npm — and does a YAML allowlist feel like the right trust boundary, or is per-API-key scoping (OAuth-level) the only thing that actually fixes this?

Where do you draw the line between sandbox and gateway?

Context: I've been building Warden (github.com/Prof-bilal/Warden) — a sandbox runtime that uses a YAML policy (filesystem/network/env) enforced at the OS level. Curious how others think about the right granularity for MCP security.

Honest limitations: proxy is HTTP/HTTPS only, no wildcards, no CPU limits — all documented.
```

---

## 4. AMA Post — Week 3 (Sept 22, 10 AM IST)

**Title:**
```
[AMA] I built Warden — an open-source sandbox for MCP servers. Ask me anything about MCP security, sandboxing, or the tradeoffs.
```

**Body:**
```
Hey r/MCP — I built Warden (github.com/Prof-bilal/Warden), a sandbox runtime for MCP servers. It enforces a YAML policy at the OS level (bwrap/Seatbelt/AppContainer).

Happy to answer anything:
- How the sandbox actually works on each OS
- Why some servers can't be sandboxed (docker, playwright)
- The tradeoffs (no wildcards, HTTP/HTTPS-only proxy, etc.)
- What breakage reports have taught me

Fire away — I'll be here for the next few hours.

Repo: github.com/Prof-bilal/Warden
Docs: prof-bilal.github.io/Warden/
```

---

## 5. Response Templates

**Technical question:**
```
Good question — [answer]. The relevant code is in [file:line] if you want to check. Full architecture: github.com/Prof-bilal/Warden/blob/main/ARCHITECTURE.md
```

**"Why not just use Docker?":**
```
Docker is one of Warden's fallback backends. But for "run one npm script with a restricted home" — namespace sandboxing has a smaller trust boundary, no daemon dependency, and near-zero startup. On Linux/macOS/Windows, Warden prefers OS-native; Docker is the fallback.
```

**"What about [specific server]?":**
```
Check the compat matrix: github.com/Prof-bilal/Warden/blob/main/docs/compatibility.md — 18 servers tested. If yours isn't listed, try `warden trace -- <your-server>` and file a report — that's the signal we want.
```

**Criticism / "This is overkill":**
```
Fair point — not every server needs sandboxing. But the default today is "full user permissions for everything", and the protocol doesn't give you a way to scope. Warden is for when you want to be explicit about what a server can touch. If the policy feels heavy, `warden trace` + `warden init` generates a starter from what the server actually does.
```

---

## 6. Posting Rules & Tips

| Rule | Detail |
|---|---|
| One subreddit per day | Don't cross-post the same content to 4 subreddits at once — looks spammy |
| Text > link | Always include context — Reddit hates bare links |
| Reply to every comment | Within 4 hours, especially first 2 hours |
| Don't be defensive | Acknowledge limitations — Reddit rewards humility |
| No hashtags | Reddit hates hashtags |
| Flair | Use `Showcase` / `Tool` / `Project` if available |
| Timing | 10 AM IST = 4:30 AM UTC = evening US — good overlap; also try 6 PM IST for US morning |
| Cross-post | After 24h, you can cross-post to one more relevant subreddit with "x-post from r/MCP" note |

## 7. Image / Video Usage

| Post | Asset | Dimensions | When to Attach |
|---|---|---|---|
| Main technical post | `warden-post-reddit-1200x630.png` (optional) | 1200×630 | Attach one image max — Reddit prefers text |
| AMA | No image | — | Text only |
| Discussion question | No image | — | Text only |

Reddit is text-first. Images are optional and should be simple (terminal screenshot or policy card). Don't over-design.
