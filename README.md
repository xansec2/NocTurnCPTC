# NocTurnCPTC

**NocTurnCPTC** is a command console for offensive-security work: a curated catalog of
pentest and bug-bounty tools whose command lines you assemble by picking a tool,
toggling its flags, and watching the command build live — then **yank** the
finished command (`ctrl+y`) and run it in a separate terminal.

It is a **terminal UI** (TUI) built with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

**NocTurnCPTC never executes anything.** It only assembles command lines for you
to copy; you decide where and whether to run them.

> This is a trimmed build of NocTurnPen with the AI output evaluator, the
> `--web` browser interface, and the in-TUI command runner removed. All that
> remains is the command builder.

## The TUI

A single full-width panel shows either:

- **the tool picker** — switch between catalogs (Setup, Footprinting, Web, AD,
  Linux, Reverse Shells, File Transfers) with `H` / `L`, filter with `/`, and pick
  a tool with `enter`.
- **the flag configurator** — toggle and fill the selected tool's flags while the
  assembled command builds live in the box at the bottom. It is modal like vim:
  it opens in normal mode, press `i` to edit a value and `jk` or `esc` to stop.

When the command looks right, press `ctrl+y` to copy it to your clipboard (or
`ctrl+o` to print it and quit), then paste it into whatever terminal you run your
tools from.

### The catalogs

NocTurnCPTC bundles several catalogs; `H` / `L` in the tool list switches between
them:

- **setup** — engagement-setup helpers (/etc/hosts, a fully-knobbed nmap).
- **Footprinting** — per-service enumeration one-liners (DNS, FTP, SMB/RPC, NFS,
  SMTP, IMAP/POP3, SNMP, SSH, MySQL, MSSQL, Oracle TNS, IPMI, RDP, WinRM, WMI,
  R-Services, RSYNC) plus crt.sh/Shodan OSINT, built from the HTBNotes vault's
  Service Enumeration notes (`footprinting_catalog.go`).
- **web** — the bug-bounty / web-recon tools and payloads documented below.
- **AD** — the Active Directory tooling from the sibling `active-directory` app
  (netexec, BloodHound, Kerberos attacks, impacket, ACL/delegation/shadow-cred
  privilege escalation, Certipy, coercion/relay, …).
- **Linux** / **Reverse Shells** — local-privesc enumeration and the reverse-shell
  payload builder.
- **File Transfers** — move files on/off a box: attacker HTTP/upload/SMB/WebDAV
  servers, Linux & Windows download cradles (wget/curl, WebClient, certutil,
  bitsadmin), uploads, netcat, and base64/openssl encoding, built from the
  vault's File Transfers notes (`filetransfer_catalog.go`).

The AD catalog itself still lives in the standalone `active-directory` TUI — here
it's a baked-in copy (`ad_catalog.go`), so re-copy that file if the AD catalog
changes.

## Tools included

Curated toward **bug bounty** — tools that find leads on their own and
complement a proxy like Caido/Burp rather than duplicating it. Grouped by
workflow stage (filter with `/` by stage name, e.g. `recon`):

- **recon** — `subfinder` · `amass` · `dnsx`
- **probe** — `httpx` · `naabu` · `gowitness`
- **crawl** — `katana` · `gau` · `waybackurls`
- **content discovery** — `ffuf` · `ffuf · token` (brute-force reset
  tokens/OTPs/IDs with regex match/filter) · `arjun` · `gobuster`
- **vuln scan** — `nuclei` · `dalfox` · `subzy` · `sqlmap` · `nikto`
- **shells** — GTFOBins reverse / bind / spawn shells (`bash`, `nc`, `socat`,
  `python`, `perl`, `php`, `ruby`, `openssl`, `telnet`, `lua`, `node`, `gawk`,
  `busybox`, `zsh`) for post-exploitation on hosts you're authorized to test
- **payloads** — a `PayloadsAllTheThings`-style cheat sheet of canonical
  detection probes (XSS, SQLi, SSTI, SSRF, path traversal, XXE, command
  injection, open redirect, NoSQL/LDAP/CRLF, JWT). Filter with `/payload`,
  fill the `{MARKER}`/`{HOST}`/`{FILE}`/`{CMD}` placeholder, then `ctrl+y` to
  copy the probe into Caido/Burp. See [Payloads](#payloads) below.

Each exposes a curated set of flags with sensible defaults, descriptions, and
required-field hints.

The **shells** branch is lifted from [GTFOBins](https://gtfobins.github.io).
Entries are named `<binary> · <technique>`; pick one, fill the `{LHOST}` /
`{LPORT}` placeholders (which are substituted straight into the payload), then
`ctrl+y` to copy. The plain `· shell` spawns take no parameters.

A typical chain: `subfinder` → `httpx` → `katana`/`gau` → `ffuf`/`arjun` →
`nuclei`, then pivot into Caido for manual testing.

## Wordlists

A curated slice of [SecLists](https://github.com/danielmiessler/SecLists) ships
in `wordlists/` so the fuzzing commands reference lists that exist out of the box
— no system install needed:

| Dir | Lists | Used by |
|-----|-------|---------|
| `web-content/` | `common.txt`, `raft-medium-directories.txt`, `raft-medium-files.txt`, `big.txt`, `quickhits.txt`, `DirBuster-2007_directory-list-2.3-medium.txt`, `raft-large-directories.txt`, `raft-large-files.txt` | `ffuf`, `gobuster` |
| `dns/` | `subdomains-top1million-{5000,20000,110000}.txt`, `bitquark-subdomains-top100000.txt` | subdomain brute / vhost fuzzing |
| `usernames/` | `top-usernames-shortlist.txt`, `cirt-default-usernames.txt` | auth fuzzing |
| `passwords/` | `darkweb2017_top-1000.txt`, `probable-v2_top-{1575,12000}.txt`, `Pwdb_top-100000.txt`, `top-passwords-shortlist.txt` | auth fuzzing |

`ffuf` and `gobuster` default their `-w` flag to a bundled list (editable in the
UI). Paths resolve to an absolute location next to the binary, so a command
copied from the TUI works from any directory you paste it into. To use a
different collection, set `PENTEST_TUI_WORDLISTS` to a directory that mirrors the
same `web-content/`, `dns/`, … sub-layout, and the defaults follow it; otherwise
just edit the `-w` path in the UI.

## Payloads

A curated, **detection-first** payload library ships in `payloads.go`, modelled
on [PayloadsAllTheThings](https://github.com/swisskyrepo/PayloadsAllTheThings)
and the [PortSwigger Web Security Academy](https://portswigger.net/web-security)
cheat sheets. Each entry is a well-known *probe* — the minimal input that
confirms whether a sink is reachable and how your input is treated — not a
weaponised exploit. They live in the same picker as the tools (filter with
`/payload`), grouped by class:

| Class | Example probes |
|-------|----------------|
| `xss` | script canary, img/svg handlers, attribute breakout, `javascript:` URI, polyglot, DOM hashprobe |
| `sqli` | quote probe, auth-bypass tautology, UNION column count/select, boolean & time-blind (MySQL/Postgres/MSSQL) |
| `sqli-build` | **composable builders** — pick a quote/paren context, comment style, DBMS primitive and column list, and the injection assembles live (see [SQLi builders](#sqli-builders)) |
| `ssti` | `${7*7}{{7*7}}` math detector, Jinja2 & Freemarker markers |
| `path-traversal` | `../`, URL- & double-encoded, PHP `filter` wrapper |
| `ssrf` | localhost, cloud metadata (`169.254.169.254`), decimal-IP bypass, OOB canary |
| `cmd-injection` | separators, command substitution, blind OOB |
| `xxe` | external-entity file read, OOB exfil |
| `privesc` | access-control probes — IDOR/BOLA id swap, mass-assignment role field, forced browsing to admin, 403-bypass rewrite headers, method-override, JWT claim tamper |
| `open-redirect`, `nosqli`, `ldapi`, `crlf`, `host-header`, `jwt` | the usual one-liners |

Placeholders are substituted straight into the copied string (same mechanism as
the GTFOBins shells):

| Token | Meaning |
|-------|---------|
| `{MARKER}` | a unique canary you grep responses for to see where input lands |
| `{HOST}` | your out-of-band host (Interactsh / Burp Collaborator) for blind detection |
| `{FILE}` | a file path target for traversal / LFI probes |
| `{CMD}` | a benign command (a timed `sleep`) to prove execution |
| `{ID}`, `{FIELD}`, `{PATH}`, `{VERB}`, `{CLAIM}`, `{VALUE}` | access-control knobs — the victim object id, privilege attribute, admin path, smuggled HTTP verb, and JWT role claim/value |

Workflow: pick a probe → fill the placeholder → `ctrl+y` to copy → paste into
your proxy and fire the request against a target **you are authorized to test**.
A payload entry's "command" is the bare payload string itself, so `ctrl+y`
(copy), `ctrl+o`, and the on-quit print all give you exactly the text to paste
into your proxy — nothing to strip off.

### SQLi builders

The `sqli-build` entries go a step past the fixed one-liners: instead of copying
a static string, you **assemble** an injection for the exact sink you're testing
by cycling a few configurable knobs (`←/→` on choice flags, type on the rest).
The command rebuilds live, and `ctrl+y` copies the finished payload.

| Builder | What you compose |
|---------|------------------|
| `build: auth bypass` | `' OR 1=1 --` tautology with your chosen quote/paren context |
| `build: boolean blind` | `AND <cond>` where `{COND}` flips between a true/false test |
| `build: time blind` | a DBMS timing primitive (MySQL `SLEEP`, Postgres `pg_sleep`, MSSQL `WAITFOR`, Oracle `RECEIVE_MESSAGE`) + delay |
| `build: UNION extract` | `UNION SELECT` with an editable column list sized to the `ORDER BY` count |
| `build: error-based extract` | leaks a subquery through a DBMS error (MySQL `extractvalue`/`updatexml`, Postgres `CAST`, MSSQL `CONVERT`) |
| `build: stacked query` | a second statement after `;` for drivers that allow it |

The shared knobs:

| Token | Flag | Meaning |
|-------|------|---------|
| `{Q}` | Break context | what closes the original value: `'`, `"`, `')`, `")`, `'))` |
| `{C}` | Comment | trailing comment that swallows the rest of the query (`-- -`, `#`, `/*`) |
| `{COLS}` | UNION columns | the `UNION SELECT` list — match the column count, carry data in a text column |
| `{SLEEP}`/`{T}` | Delay primitive / seconds | DBMS sleep function and how long |
| `{COND}` | Condition | the boolean oracle (`1=1` vs `1=2`, or a sample extraction test) |
| `{ERR}`/`{SUB}` | Error primitive / Leak subquery | the error-cast and the subquery it surfaces |
| `{STMT}` | Stacked statement | the statement run after `;` |

The assembled command is the bare SQL payload — `ctrl+y` copies exactly what you
paste into your proxy (quotes and all), with no wrapper to strip.

## Build & run

Go 1.27+ is required. This repo was set up with Go installed to `~/.local/share/go-sdk/go`,
so either run `source ~/.bashrc` (adds it to PATH) or let `run.sh` find it:

```bash
cd NocTurnCPTC
go build -o nocturncptc .   # pure-Go build; CGO is off (see run.sh)
./nocturncptc               # launch the TUI
```

Or just `./run.sh` — it finds (or installs) Go, rebuilds, and launches.

To install it system-wide (binary at `/opt/nocturncptc`, launcher `nocturncptc`
on every user's PATH), run `sudo ./install.sh --build` from inside the app
directory.

## Keys

### Tool list
| Key | Action |
|-----|--------|
| `j` / `k` | move |
| `h` / `l`, `ctrl+u` / `ctrl+d` | previous / next page |
| `g` / `G` | first / last tool |
| `/` | filter (`jk` or `enter` keeps the filter, `esc` cancels it) |
| `enter` | configure selected tool |
| `H` / `L` | previous / next catalog |
| `q` / `ctrl+c` | quit |

### Flag configurator: normal mode
| Key | Action |
|-----|--------|
| `j` / `k` | move between flags |
| `gg` / `G` | first / last flag |
| `space` | toggle a boolean flag |
| `h` / `l` | cycle a choice flag, or move the cursor in a text/number flag |
| `0` / `$`, `w` / `b` | line start / end, word forward / back |
| `i` / `a`, `I` / `A` | insert at / after the cursor, at line start / end |
| `x`, `D`, `dd` | delete the character, to the end of the value, the whole value |
| `C`, `cc` or `S` | change to the end of the value, the whole value |
| `enter` | toggle a boolean, cycle a choice, or edit a text/number flag |
| `y` or `ctrl+y` | copy the command to the clipboard |
| `ctrl+o` | print the command and quit |
| `q` / `esc` | back to the tool list |
| `ctrl+c` | quit |

### Flag configurator: insert mode
| Key | Action |
|-----|--------|
| *(type)* | edit the selected text/number flag |
| `jk` or `esc` | back to normal mode |
| `↑/↓` or `tab` | back to normal mode and move between flags |
| `ctrl+y` / `ctrl+o` / `ctrl+c` | copy / print and quit / quit |

`jk` needs `k` within 500ms of `j`, so to type those letters literally, pause
between them. Arrow keys and `tab` / `shift+tab` keep working wherever they did.

The last assembled command is also printed to stdout when you quit (via `ctrl+o`
or after a `ctrl+y` copy), so it survives leaving the TUI even if the clipboard
is unavailable (e.g. over SSH without an X clipboard bridge).

## Adding a tool

Edit `catalog.go` and append a `Tool` to the `catalog` slice. Each `Flag` has a
`Kind` (`KindBool`, `KindString`, `KindInt`, `KindChoice`), an optional default
(`Value` / `Enabled`), a `Desc`, and flags like `Required` or `Positional`.
Positional flags (targets, URLs with no switch) are emitted at the end of the
command.

## Files

| File | Purpose |
|------|---------|
| `tool.go` | `Tool`/`Flag` types + command assembly and shell quoting |
| `catalog.go` | the built-in web tool definitions |
| `ad_catalog.go` | the Active Directory catalog (baked-in copy of the `active-directory` app's), served as the **AD** catalog |
| `footprinting_catalog.go` | per-service enumeration one-liners (**Footprinting** catalog) |
| `filetransfer_catalog.go` | file-transfer servers, cradles, and encoders (**File Transfers** catalog) |
| `linux_catalog.go` | Linux local-privesc enumeration (**Linux** catalog) |
| `revshell_catalog.go` | the reverse-shell payload builder (**Reverse Shells** catalog) |
| `setup_catalog.go` | engagement-setup helpers (**setup** catalog) |
| `payloads.go` | the detection-probe payload library (appended into the catalog) |
| `catalogs.go` | the catalog registry (which catalogs the TUI offers) |
| `model.go` | Bubble Tea model, update loop, and rendering |
| `wordlists.go` | resolves the bundled SecLists paths |
| `main.go` | entrypoint; prints the last assembled command on quit |

## Safety

NocTurnCPTC only **builds** command lines — it never runs them. You copy a
command and run it yourself, and nothing checks a command's target for you. Run
the tools it assembles only against systems you are authorized to test.
# NocTurnCPTC
