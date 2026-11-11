#!/usr/bin/env bash
# Install PhantomDeck system-wide so every user on this box can run `phantomdeck`.
#
# Layout:
#   /opt/phantomdeck/phantomdeck   the real binary (rebuilt below with --build)
#   /opt/phantomdeck/wordlists     bundled wordlists, found next to the binary
#   /usr/local/bin/phantomdeck     global launcher (on everyone's PATH)
#
# Run from inside the app directory on the target box:
#   sudo ./install.sh            # use the prebuilt ./phantomdeck binary
#   sudo ./install.sh --build    # rebuild from source first (needs Go 1.27+)
set -euo pipefail

SRC="$(cd "$(dirname "$0")" && pwd)"
APP_DIR=/opt/phantomdeck

if [ "$(id -u)" -ne 0 ]; then
	echo "install.sh: run me with sudo (needs to write /opt and /usr/local/bin)." >&2
	exit 1
fi

# Optionally rebuild so the binary matches this box exactly.
if [ "${1:-}" = "--build" ]; then
	if [ -x "$HOME/.local/share/go-sdk/go/bin/go" ]; then
		export PATH="$HOME/.local/share/go-sdk/go/bin:$PATH"
	fi
	if ! command -v go >/dev/null 2>&1; then
		echo "install.sh: --build given but 'go' not found on PATH." >&2
		exit 1
	fi
	echo "==> building phantomdeck from source"
	( cd "$SRC" && CGO_ENABLED=0 go build -o phantomdeck . )
fi

if [ ! -f "$SRC/phantomdeck" ]; then
	echo "install.sh: no 'phantomdeck' binary in $SRC (run with --build, or build it first)." >&2
	exit 1
fi

echo "==> installing app into $APP_DIR"
install -d -m 0755 "$APP_DIR"
install -m 0755 "$SRC/phantomdeck" "$APP_DIR/phantomdeck"
rm -rf "$APP_DIR/wordlists"
cp -a "$SRC/wordlists" "$APP_DIR/wordlists"
# The bundled wordlists are redistributed from SecLists (MIT); see
# wordlists/ATTRIBUTION.md. If you would rather NOT redistribute them, delete the
# cp above and fetch SecLists at install time instead, e.g.:
#   git clone --depth 1 https://github.com/danielmiessler/SecLists "$APP_DIR/wordlists"
# The app resolves wordlists by path at runtime, so either approach works.
# World-readable files, traversable dirs; nothing group/other-writable.
chmod -R a+rX "$APP_DIR"
chmod 0755 "$APP_DIR/phantomdeck"

echo "==> writing global launcher /usr/local/bin/phantomdeck"
cat > /usr/local/bin/phantomdeck <<'EOF'
#!/usr/bin/env bash
# Global launcher for PhantomDeck. The app only assembles command lines; you
# run the commands you yank (ctrl+y) in your own terminal.
exec /opt/phantomdeck/phantomdeck "$@"
EOF
chmod 0755 /usr/local/bin/phantomdeck

echo
echo "Done. Any user can now run:"
echo "    phantomdeck    # terminal UI — yank commands with ctrl+y"
