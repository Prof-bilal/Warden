package mcpserver

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// timeNow is a var so tests can pin run IDs.
var timeNow = time.Now

// WidgetResources lists the MCP Apps resources this server publishes. Three
// focused views only: policy, execution, trace. No dashboard.
func WidgetResources() []map[string]any {
	desc := func(s string) string { return s }
	return []map[string]any{
		{
			"uri":         "ui://warden/policy",
			"name":        "Warden Policy View",
			"description": desc("Structured policy view: allowed filesystem paths, network hosts, and env vars, with deny-by-default markers."),
			"mimeType":    "text/html",
		},
		{
			"uri":         "ui://warden/execution",
			"name":        "Warden Execution View",
			"description": desc("Sandboxed run summary: status, allowed/blocked operation counts, and a link to the trace."),
			"mimeType":    "text/html",
		},
		{
			"uri":         "ui://warden/trace",
			"name":        "Warden Trace View",
			"description": desc("Audit trace: chronological access attempts with allowed/denied markers."),
			"mimeType":    "text/html",
		},
	}
}

func widgetHTML(uri string) (string, bool) {
	switch uri {
	case "ui://warden/policy":
		return policyWidgetHTML, true
	case "ui://warden/execution":
		return executionWidgetHTML, true
	case "ui://warden/trace":
		return traceWidgetHTML, true
	}
	return "", false
}

// The widgets are self-contained HTML documents that read structuredContent
// from the host (MCP Apps / OpenAI Apps SDK postMessage bridge) and render a
// compact, dark, developer-tool view. Green = allowed, red = denied.
// They never display secret values — only resource names and decisions.
const baseCSS = `
  :root { color-scheme: dark; }
  * { box-sizing: border-box; margin: 0; }
  body {
    font: 13px/1.5 ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
    background: #0d1117; color: #c9d1d9; padding: 16px;
  }
  .hdr { display: flex; align-items: center; gap: 8px; margin-bottom: 12px; }
  .brand { font-weight: 700; letter-spacing: 0.12em; color: #58a6ff; }
  .tag { font-size: 11px; color: #8b949e; }
  .card { background: #161b22; border: 1px solid #30363d; border-radius: 8px; padding: 12px; margin-bottom: 10px; }
  .card h2 { font-size: 11px; text-transform: uppercase; letter-spacing: 0.1em; color: #8b949e; margin-bottom: 8px; }
  .row { display: flex; gap: 8px; padding: 2px 0; align-items: baseline; }
  .ok { color: #3fb950; } .no { color: #f85149; } .dim { color: #8b949e; }
  .pill { display: inline-block; padding: 1px 8px; border-radius: 10px; font-size: 11px; }
  .pill.ok { background: #12261e; } .pill.no { background: #2d1417; }
  .stat { display: inline-block; margin-right: 16px; }
  .stat b { font-size: 18px; display: block; }
  .note { margin-top: 8px; font-size: 11px; color: #8b949e; }
`

func wrapWidget(title string, script string) string {
	return "<!DOCTYPE html>\n<html>\n<head>\n<meta charset=\"utf-8\">\n<title>" + html.EscapeString(title) +
		"</title>\n<style>" + baseCSS + "</style>\n</head>\n<body>\n" +
		"<div class=\"hdr\"><span class=\"brand\">WARDEN</span><span class=\"tag\">" + html.EscapeString(title) + "</span></div>\n" +
		"<div id=\"app\"><span class=\"dim\">Waiting for data…</span></div>\n" +
		"<script>" + script + "</script>\n</body>\n</html>\n"
}

var policyWidgetHTML = wrapWidget("Policy", `
(function () {
  function esc(s) { var d = document.createElement('div'); d.textContent = s == null ? '' : String(s); return d.innerHTML; }
  function getData() { return window.structuredContent || (window.parent !== window ? {} : {}); }
  function render(data) {
    var app = document.getElementById('app');
    if (!data || (!data.filesystem && !data.network && !data.env)) {
      app.innerHTML = '<span class="dim">No policy data.</span>'; return;
    }
    var html = '';
    function section(name, allows, note, sub) {
      html += '<div class="card"><h2>' + esc(name) + '</h2>';
      (allows || []).forEach(function (a) {
        var label = esc(typeof a === 'string' ? a : JSON.stringify(a));
        html += '<div class="row"><span class="ok">✓</span><span>' + label + (sub ? ' <span class="dim">' + esc(sub) + '</span>' : '') + '</span></div>';
      });
      html += '<div class="row"><span class="no">✗</span><span class="dim">' + esc(note) + '</span></div>';
      html += '</div>';
    }
    section('Filesystem', [].concat(data.filesystem && data.filesystem.read || []).map(function (p) { return p + ' (read)'; })
      .concat((data.filesystem && data.filesystem.write || []).map(function (p) { return p + ' (read-write)'; })),
      data.filesystem && data.filesystem.denied_note || 'everything else is denied');
    section('Network', data.network && data.network.allow, data.network && data.network.denied_note || 'all other hosts blocked');
    section('Environment', data.env && data.env.allow, data.env && data.env.denied_note || 'all other variables withheld');
    if (data.limits && (data.limits.memory_mb > 0 || data.limits.timeout_s > 0)) {
      html += '<div class="card"><h2>Limits</h2><div class="row dim">';
      if (data.limits.memory_mb > 0) html += '<span>memory_mb: ' + data.limits.memory_mb + '</span>';
      if (data.limits.timeout_s > 0) html += '<span>timeout_s: ' + data.limits.timeout_s + '</span>';
      html += '</div></div>';
    }
    if (data.policy_path) html += '<div class="note">Policy: ' + esc(data.policy_path) + '</div>';
    html += '<div class="note">Deny by default — anything not listed is blocked.</div>';
    app.innerHTML = html;
  }
  window.addEventListener('message', function (e) {
    var m = e.data || {};
    if (m.jsonrpc === 'ui/notifications/initialized' || m.method === 'ui/notifications/initialized') {
      render(getData());
    }
    if (m && m.result && m.result.structuredContent) render(m.result.structuredContent);
    if (m && m.structuredContent) render(m.structuredContent);
  });
  // Some hosts inject structured content directly.
  if (window.structuredContent) render(window.structuredContent);
})();
`)

var executionWidgetHTML = wrapWidget("Execution", `
(function () {
  function esc(s) { var d = document.createElement('div'); d.textContent = s == null ? '' : String(s); return d.innerHTML; }
  function render(r) {
    var app = document.getElementById('app');
    if (!r) { app.innerHTML = '<span class="dim">No execution data.</span>'; return; }
    var ok = r.status === 'completed';
    var html = '<div class="card"><h2>Status</h2>' +
      '<span class="pill ' + (ok ? 'ok">✓ ' : 'no">✗ ') + esc(r.status || 'unknown') + '</span>';
    if (typeof r.exit_code === 'number') html += ' <span class="dim">exit code ' + r.exit_code + '</span>';
    if (r.backend) html += '<div class="row"><span class="dim">backend: ' + esc(r.backend) + '</span></div>';
    html += '<div class="row"><span class="dim">policy: ' + esc(r.policy_path || '') + '</span></div>';
    html += '<div class="row"><span class="dim">command: ' + esc((r.command || []).join(' ')) + '</span></div>';
    html += '</div>';
    html += '<div class="card"><h2>Activity</h2>' +
      '<span class="stat"><b class="ok">' + (r.allowed_ops || 0) + '</b><span class="dim">allowed</span></span>' +
      '<span class="stat"><b class="no">' + (r.denied_ops || 0) + '</b><span class="dim">blocked</span></span>';
    if (r.denied_sample && r.denied_sample.length) {
      html += '<div class="row no" style="margin-top:8px">✗ Denied:</div>';
      r.denied_sample.forEach(function (d) { html += '<div class="row"><span class="no">✗</span><span>' + esc(d) + '</span></div>'; });
    }
    html += '</div>';
    if (r.run_id) html += '<div class="note">run_id: ' + esc(r.run_id) + ' — ask to trace this execution for details.</div>';
    app.innerHTML = html;
  }
  window.addEventListener('message', function (e) {
    var m = e.data || {};
    if (m && m.result && m.result.structuredContent && m.result.structuredContent.run) render(m.result.structuredContent.run);
    if (m && m.structuredContent && m.structuredContent.run) render(m.structuredContent.run);
  });
  if (window.structuredContent && window.structuredContent.run) render(window.structuredContent.run);
})();
`)

var traceWidgetHTML = wrapWidget("Trace", `
(function () {
  function esc(s) { var d = document.createElement('div'); d.textContent = s == null ? '' : String(s); return d.innerHTML; }
  function render(t) {
    var app = document.getElementById('app');
    if (!t) { app.innerHTML = '<span class="dim">No trace data.</span>'; return; }
    var html = '<div class="card"><h2>Summary</h2>' +
      '<div class="row"><span class="stat"><b>' + (t.total_events || 0) + '</b><span class="dim">events</span></span>' +
      '<span class="stat"><b class="ok">' + (t.allowed || 0) + '</b><span class="dim">allowed</span></span>' +
      '<span class="stat"><b class="no">' + (t.denied || 0) + '</b><span class="dim">denied</span></span></div>' +
      '<div class="row dim">' + (t.filesystem_ops || 0) + ' filesystem · ' + (t.network_ops || 0) + ' network</div></div>';
    if (t.denied_sample && t.denied_sample.length) {
      html += '<div class="card"><h2>Denied</h2>';
      t.denied_sample.forEach(function (d) { html += '<div class="row"><span class="no">✗</span><span>' + esc(d) + '</span></div>'; });
      html += '</div>';
    }
    if (t.log_path) html += '<div class="note">log: ' + esc(t.log_path) + '</div>';
    app.innerHTML = html;
  }
  window.addEventListener('message', function (e) {
    var m = e.data || {};
    if (m && m.result && m.result.structuredContent) render(m.result.structuredContent);
    if (m && m.structuredContent) render(m.structuredContent);
  });
  if (window.structuredContent) render(window.structuredContent);
})();
`)

// unusedFmt keeps fmt imported for widget string building helpers.
var _ = fmt.Sprintf
var _ = strings.TrimSpace
