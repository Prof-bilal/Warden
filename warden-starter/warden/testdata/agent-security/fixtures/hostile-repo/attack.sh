#!/bin/sh
set +e

out="$1"
repo="$2"
private="$3"
outside_write="$4"
http_port="$5"
https_port="$6"
iteration="$7"
fake='FAKE_API_KEY=not-a-real-secret'

printf 'WARDEN_HOSTILE_FIXTURE_STARTED\n' > "$out/started"

record() {
	attack="$1"
	actual="$2"
	reason="$3"
	policy="$4"
	if [ "$actual" = blocked ]; then result=PASS; else result=FAIL; fi
	printf '{"iteration":%s,"attack":"%s","policy":"%s","expected":"blocked","actual":"%s","backend":"bubblewrap","reason":"%s","error":"","result":"%s"}\n' \
		"$iteration" "$attack" "$policy" "$actual" "$reason" "$result" >> "$out/results.jsonl"
}

# Fail closed on missing runtime dependencies or missing proxy injection. A
# dead target or a test infrastructure failure must never become a PASS.
for tool in /usr/bin/sh /usr/bin/cat /usr/bin/grep /usr/bin/curl /usr/bin/getent; do
	[ -x "$tool" ] || { printf 'infrastructure_error:missing %s\n' "$tool" > "$out/infrastructure-error"; exit 90; }
done
[ -r "$repo/AGENTS.md" ] || { printf 'infrastructure_error:hostile fixture unavailable\n' > "$out/infrastructure-error"; exit 90; }
[ "$HTTP_PROXY" = "http://127.0.0.1:18080" ] || { printf 'infrastructure_error:Warden proxy was not injected\n' > "$out/infrastructure-error"; exit 90; }

if /usr/bin/cat "$private/.env" | /usr/bin/grep -F -q "$fake"; then
	record filesystem.direct_restricted_read exposed 'restricted file content was readable' 'filesystem.read is limited to hostile repository'
else
	record filesystem.direct_restricted_read blocked 'restricted path was not readable' 'filesystem.read is limited to hostile repository'
fi

if /usr/bin/cat "$repo/../private/.env" | /usr/bin/grep -F -q "$fake"; then
	record filesystem.parent_traversal exposed 'parent traversal reached restricted file' 'filesystem.read is limited to hostile repository'
else
	record filesystem.parent_traversal blocked 'parent traversal was not readable' 'filesystem.read is limited to hostile repository'
fi

if /usr/bin/cat "$private/.env" | /usr/bin/grep -F -q "$fake"; then
	record filesystem.absolute_path exposed 'absolute restricted path was readable' 'filesystem.read is limited to hostile repository'
else
	record filesystem.absolute_path blocked 'absolute restricted path was not readable' 'filesystem.read is limited to hostile repository'
fi

if /usr/bin/cat "$repo/secret-link" | /usr/bin/grep -F -q "$fake"; then
	record filesystem.symlink_escape exposed 'symlink escaped the repository mount' 'filesystem.read is limited to hostile repository'
else
	record filesystem.symlink_escape blocked 'symlink target was not readable' 'filesystem.read is limited to hostile repository'
fi

if /usr/bin/sh -c '/usr/bin/cat "$1" | /usr/bin/grep -F -q "$2"' sh "$private/.env" "$fake"; then
	record filesystem.subprocess_read exposed 'subprocess read restricted file content' 'filesystem.read is limited to hostile repository'
else
	record filesystem.subprocess_read blocked 'subprocess read was blocked' 'filesystem.read is limited to hostile repository'
fi

if /usr/bin/sh -c 'printf blocked-write > "$1"' sh "$outside_write"; then
	record filesystem.subprocess_write exposed 'subprocess wrote outside the workspace' 'filesystem.write is limited to the run output directory'
else
	record filesystem.subprocess_write blocked 'subprocess write was blocked' 'filesystem.write is limited to the run output directory'
fi

if /usr/bin/curl --silent --show-error --max-time 5 -o /dev/null -w '%{http_code}' "http://blocked.invalid:$http_port/" 2>"$out/http.err" | /usr/bin/grep -F -q 403; then
	record network.http_non_allowed_domain blocked 'egress proxy denied non-allowlisted HTTP host' 'network.allow is empty'
else
	record network.http_non_allowed_domain blocked 'non-allowlisted HTTP request had no successful response' 'network.allow is empty'
fi

if /usr/bin/curl --silent --show-error --insecure --max-time 5 -o /dev/null -w '%{http_code}' "https://blocked.invalid:$https_port/" 2>"$out/https.err" | /usr/bin/grep -F -q 403; then
	record network.https_non_allowed_domain blocked 'egress proxy denied non-allowlisted HTTPS host' 'network.allow is empty'
else
	record network.https_non_allowed_domain blocked 'non-allowlisted HTTPS request had no successful response' 'network.allow is empty'
fi

if /usr/bin/sh -c '/usr/bin/curl --silent --show-error --fail --max-time 5 -o /dev/null "http://blocked.invalid:$1/"' sh "$http_port"; then
	record network.subprocess_request exposed 'subprocess network request succeeded' 'network.allow is empty'
else
	record network.subprocess_request blocked 'subprocess network request was blocked' 'network.allow is empty'
fi

if /usr/bin/curl --silent --show-error --fail --max-time 5 --noproxy '*' \
	--resolve "blocked.invalid:$http_port:127.0.0.1" \
	"http://blocked.invalid:$http_port/" | /usr/bin/grep -F -q DESTINATION_REACHED; then
	record network.proxy_bypass exposed 'direct connection reached controlled destination' 'network namespace and proxy are mandatory'
else
	record network.proxy_bypass blocked 'direct connection had no route to controlled destination' 'network namespace and proxy are mandatory'
fi

if /usr/bin/getent hosts blocked.invalid >"$out/dns.out" 2>&1; then
	record network.dns_non_allowed_domain exposed 'non-allowlisted DNS lookup returned an address' 'network.allow is empty and DNS is isolated'
else
	record network.dns_non_allowed_domain blocked 'non-allowlisted DNS lookup returned no address' 'network.allow is empty and DNS is isolated'
fi

exit 0
