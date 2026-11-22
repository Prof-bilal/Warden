import { spawn } from "node:child_process";

export function terminal(bin, args, replies, { expectedExit = 0, env = process.env } = {}) {
  // util-linux script provides the controlling terminal that Warden requires.
  const quote = word => "'" + word.replaceAll("'", "'\\''") + "'";
  return new Promise((resolve, reject) => {
    const child = spawn("script", ["-qefc", [bin, ...args].map(quote).join(" "), "/dev/null"], { env, stdio: ["pipe", "pipe", "pipe"] });
    let output = "", pending = "";
    child.stdout.on("data", data => {
      output += data;
      pending += data;
      const next = replies[0];
      if (next && pending.includes(next[0])) {
        pending = pending.slice(pending.indexOf(next[0]) + next[0].length);
        child.stdin.write(replies.shift()[1] + "\n");
      }
    });
    child.stdin.on("error", () => {}); // The process may exit immediately after a response.
    child.stderr.on("data", data => { output += data; });
    child.on("error", reject);
    child.on("close", code => code === expectedExit && replies.length === 0
      ? resolve(output)
      : reject(Error(`terminal exited ${code}; expected ${expectedExit}; unanswered prompts ${replies.length}\n${output}`)));
  });
}
