#!/usr/bin/env bash
# Spin up a CPTC tmux session: `tool` launches PhantomDeck, plus web/linux/windows tabs.
S=phantomdeck
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Build the session only if it doesn't exist yet.
if ! tmux has-session -t "$S" 2>/dev/null; then
  # First window: tool — launch PhantomDeck when the session starts.
  tmux new-session -d -s "$S" -n tool -c "$DIR" "$DIR/run.sh"
  # Keep tool open if run.sh exits (build error, crash) so the failure stays
  # on screen instead of the window — and with it the whole session — vanishing.
  tmux set-window-option -t "$S:tool" remain-on-exit on
  tmux neww -t "$S" -n web     -c "$DIR"
  tmux neww -t "$S" -n linux   -c "$DIR"
  tmux neww -t "$S" -n windows -c "$DIR"
  tmux select-window -t "$S:tool"
fi

# Inside tmux: switch to the session. Outside tmux: attach to it.
if [ -n "$TMUX" ]; then
  tmux switch-client -t "$S"
else
  tmux attach -t "$S"
fi
