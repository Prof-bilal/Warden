#!/usr/bin/env node
/**
 * Generates per-scene narration audio with edge-tts (neural voices).
 *
 * Usage: node scripts/gen-narration.js
 * Output: video/public/audio/narration/NN-<scene>.mp3
 *
 * Each script block matches video/voiceover.md. Narration is synthesized
 * per scene so AudioTrack can drop each file exactly at the scene start.
 */
const { execSync } = require("child_process");
const fs = require("fs");
const path = require("path");

const OUT_DIR = path.join(__dirname, "..", "public", "audio", "narration");
const VOICE = process.env.NARRATION_VOICE || "en-US-AndrewMultilingualNeural";
const RATE = "+8%"; // slightly brisk to fit the timeline

const SCRIPTS = {
  "01-hook": `You've probably run a command like this. An MCP server starts. It connects. It's ready. Everything looks normal. But the process behind that server runs as you. Your files. Your environment. Your SSH keys. Your credentials. Everything in your home directory is right there. Your MCP doesn't need access to your entire computer. So why give it that access?`,
  "02-problem": `MCP, the Model Context Protocol, is why AI assistants can actually do things. It connects a client to servers that expose tools: filesystem access, network calls, environment lookups. That's what makes it useful. Here's the part worth paying attention to. Most MCP servers run as plain processes on your machine, with your permissions. Not in a container. Not behind anything. Just, running. So the same protocol that gives the model useful reach also gives the process behind it broad reach: project files, environment variables, SSH keys, credentials, your network. Useful access and unrestricted access are not the same thing. MCP gives you the first. Nothing about the protocol requires the second.`,
  "03-access": `Look at two requests from the same server. The first one reads a project file. If you asked an assistant to summarize your code, that's exactly the access it needs. Reasonable. The second one reads an SSH private key. Nothing about that request is unreasonable for a process with full filesystem access, which is the problem. The process is technically capable, so nothing stops the request. Reading a project file can be necessary. Reading a private key is sensitive. The difference isn't the protocol. The difference is where the security boundary sits, and who controls it. So, who controls that boundary today?`,
  "04-intro": `This is Warden. An open source sandbox runtime for MCP servers. The idea is simple. The MCP server runs inside a boundary made of two things: a policy, and a sandbox. Outside that boundary, your sensitive resources stay where they are. Invisible to the server. Warden gives every MCP server a controlled environment with explicit permissions. Nothing more.`,
  "05-policy": `The boundary is defined in one YAML file. This is real policy syntax. An actual example from the repository. Which directories the server may read or write. Which hosts it may connect to. Which environment variables pass through. Even resource limits. And here's the detail that matters: Warden is deny by default. Unlisted paths aren't "permission denied". They return "not found". The sandbox can't even be probed.`,
  "06-demo": `Let's run it. This is the actual CLI. Warden run, with a policy. Before launching anything, Warden prints exactly what the policy grants: the filesystem paths, the network host, the one environment variable. Then: sandbox active. From here, your MCP client talks to the server over stdio exactly as if nothing had changed. Sandboxing is invisible to the protocol. But watch the audit log. A connection to the permitted API host: allowed. A connection to an unlisted host: blocked. Every attempt, allowed or blocked, is recorded as structured JSON. Warden doctor verifies the host can actually enforce the sandbox, before you run anything.`,
  "07-architecture": `Under the hood, every request follows the same path: the MCP server asks, Warden's policy engine checks the policy, the sandbox backend enforces it at the OS level, and network egress flows through an allowlist proxy. Every decision lands in the audit log. Allowed requests reach only the resources you listed. Everything else stops at the boundary. And Warden fails closed: if the sandbox primitives can't be applied, it refuses to run your server at all. A plain process fallback is never acceptable.`,
  "08-platforms": `Warden uses the sandbox each OS already provides. On Linux, bubblewrap namespaces. On macOS, Seatbelt profiles. On Windows, AppContainer with filtering and Job Objects. And where no native backend fits, a Docker fallback: read-only root, no network. No virtual machine, no daemon required on the native path.`,
  "09-why": `Without a boundary, an MCP server reaches the host with broad permissions. With Warden, the same server passes through policy and sandbox, and touches only what you granted. Give tools the access they need. Not the access they don't.`,
  "10-cta": `Warden is open source, MIT licensed. Install it from npm, trace a server, generate a starter policy, and run it sandboxed. The repository includes the full policy schema, the CLI reference, the security model and its honest limitations, and a reproducible test harness. Clone it. Run your MCPs. Don't blindly trust them.`,
};

(async () => {
  fs.mkdirSync(OUT_DIR, { recursive: true });
  for (const [name, text] of Object.entries(SCRIPTS)) {
    const out = path.join(OUT_DIR, `${name}.mp3`);
    execSync(
      `python3 -m edge_tts --voice "${VOICE}" --rate=${RATE} --text ${JSON.stringify(text)} --write-media ${out}`,
      { stdio: "inherit" }
    );
    const kb = (fs.statSync(out).size / 1024).toFixed(0);
    console.log(`✓ ${name}.mp3 (${kb} KB)`);
  }
  console.log("\nDone. Narration in", OUT_DIR);
})();
