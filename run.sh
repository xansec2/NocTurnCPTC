#!/usr/bin/env bash
# Build (if needed) and launch the pentest TUI.
set -e
cd "$(dirname "$0")"

# Go toolchain first, so the `go build` below is found. This repo was set up
# with Go installed to ~/.local/share/go-sdk/go; only prepend that if it exists, otherwise
# fall back to whatever `go` is already on PATH.
if [ -x "$HOME/.local/share/go-sdk/go/bin/go" ]; then
	export PATH="$HOME/.local/share/go-sdk/go/bin:$PATH"
fi

# If Go still isn't available, try to install it automatically. This targets a
# Kali VM (Debian-based) but works on any Linux: it drops a fresh toolchain into
# ~/.local/share/go-sdk/go, the same path the block above looks for, so re-runs are cheap.
install_go() {
	local version="1.27.0"
	local arch
	case "$(uname -m)" in
		x86_64|amd64)  arch="amd64" ;;
		aarch64|arm64) arch="arm64" ;;
		armv6l|armv7l) arch="armv6l" ;;
		i386|i686)     arch="386" ;;
		*)
			echo "run.sh: unsupported architecture '$(uname -m)' for auto-install." >&2
			return 1
			;;
	esac

	local tarball="go${version}.linux-${arch}.tar.gz"
	local url="https://go.dev/dl/${tarball}"
	local tmp
	tmp="$(mktemp -d)"

	echo "run.sh: 'go' not found — installing Go ${version} to ~/.local/share/go-sdk/go ..." >&2

	local fetch
	if command -v curl >/dev/null 2>&1; then
		fetch="curl -fsSL -o"
	elif command -v wget >/dev/null 2>&1; then
		fetch="wget -qO"
	else
		echo "run.sh: need curl or wget to download Go (apt-get install -y curl)." >&2
		rm -rf "$tmp"
		return 1
	fi

	if ! $fetch "$tmp/$tarball" "$url"; then
		echo "run.sh: failed to download $url" >&2
		rm -rf "$tmp"
		return 1
	fi

	mkdir -p "$HOME/.local/share/go-sdk"
	rm -rf "$HOME/.local/share/go-sdk/go"
	if ! tar -C "$HOME/.local/share/go-sdk" -xzf "$tmp/$tarball"; then
		echo "run.sh: failed to extract $tarball" >&2
		rm -rf "$tmp"
		return 1
	fi

	rm -rf "$tmp"
	export PATH="$HOME/.local/share/go-sdk/go/bin:$PATH"
}

if ! command -v go >/dev/null 2>&1; then
	if ! install_go || ! command -v go >/dev/null 2>&1; then
		echo "run.sh: 'go' not found on PATH and auto-install failed." >&2
		echo "Install Go 1.27+ (or put it at ~/.local/share/go-sdk/go) and try again." >&2
		exit 1
	fi
fi

# Pure-Go build: the system gcc here
# crashes on cgo, so keep cgo off (the DNS resolver falls back to pure Go).
export CGO_ENABLED=0
go build -o nocturncptc .

# NocTurnCPTC only assembles command lines — it does not run them. Yank a
# command with ctrl+y and paste it into a separate terminal, where you are
# responsible for having the tools (subfinder, httpx, ffuf, nuclei, netexec,
# impacket-*, sqlmap, …) on your PATH.

# Optional: point ffuf/gobuster at a system SecLists install instead of the
# bundled ./wordlists dir. Uncomment and adjust if you keep SecLists elsewhere.
# export PENTEST_TUI_WORDLISTS="$HOME/tools/SecLists"

exec ./nocturncptc "$@"
