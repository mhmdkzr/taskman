package components

// pageStyles is the dashboard's CSS, kept as a Go string so templ can inline
// it. It is the same stylesheet the pre-templ version shipped, plus rules for
// markdown-rendered prose, diffs, and per-tool blocks.
func pageStyles() string {
	return `:root {
  --bg: #0b0e14;
  --bg2: #10151f;
  --bg3: #161c28;
  --border: #232b3a;
  --fg: #d5dbe6;
  --muted: #7b8698;
  --accent: #5ea1ff;
  --green: #4ec37c;
  --red: #e05561;
  --yellow: #d9a94e;
  --purple: #b48add;
  --mono: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}
* { box-sizing: border-box; }
body {
  margin: 0;
  background: var(--bg);
  color: var(--fg);
  font: 13px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
}
a { color: var(--accent); text-decoration: none; }

/* kanban board */
.board {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  height: 100vh;
  padding: 12px 16px;
}
.col {
  display: flex;
  flex-direction: column;
  background: var(--bg2);
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  min-width: 0;
}
.col-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border-bottom: 1px solid var(--border);
  background: var(--bg3);
}
.col-title { font-size: 11px; text-transform: uppercase; letter-spacing: 0.08em; font-weight: 600; color: var(--muted); }
.col-count {
  font-size: 11px; font-family: var(--mono); color: var(--muted);
  background: var(--bg); border: 1px solid var(--border); border-radius: 10px; padding: 0 6px;
}
.col-body { overflow-y: auto; padding: 8px; display: flex; flex-direction: column; gap: 8px; flex: 1; }

.task {
  border: 1px solid var(--border); border-radius: 6px; padding: 8px 10px;
  background: var(--bg3); cursor: pointer;
}
.task:hover { border-color: var(--accent); }
.task.active { border-color: var(--accent); box-shadow: 0 0 0 1px var(--accent); }
.task .row { display: flex; align-items: center; gap: 8px; }
.task .title { font-weight: 600; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.task .meta { color: var(--muted); font-size: 11px; margin-top: 4px; font-family: var(--mono); }
.task .phase { color: var(--accent); font-size: 11px; margin-top: 4px; }
.badge {
  display: inline-block; padding: 1px 7px; border-radius: 10px;
  font-size: 10px; font-weight: 600; text-transform: uppercase; letter-spacing: 0.04em;
  background: var(--bg3); border: 1px solid var(--border); color: var(--muted);
}
.badge.created { color: var(--muted); }
.badge.started { color: var(--yellow); border-color: var(--yellow); }
.badge.completed { color: var(--purple); border-color: var(--purple); }
.badge.reviewed { color: var(--green); border-color: var(--green); }
.badge.failed { color: var(--red); border-color: var(--red); }
.empty { color: var(--muted); padding: 16px; text-align: center; }

/* detail drawer */
.backdrop {
  position: fixed;
  top: 0; right: 0; bottom: 0; left: 0;
  background: rgba(0,0,0,0.5);
  z-index: 10;
}
.drawer {
  position: fixed;
  top: 0; right: 0; bottom: 0;
  width: 560px;
  max-width: 100vw;
  background: var(--bg2);
  border-left: 1px solid var(--border);
  z-index: 20;
  overflow-y: auto;
  padding: 16px;
  box-shadow: -8px 0 24px rgba(0,0,0,0.4);
}
.drawer .close {
  position: absolute; top: 10px; right: 12px;
  background: var(--bg3); color: var(--muted); border: 1px solid var(--border);
  border-radius: 6px; padding: 4px 10px; cursor: pointer; font-size: 12px;
}
.drawer .close:hover { color: var(--fg); border-color: var(--accent); }
.detail { max-width: 900px; }
.detail h2 { font-size: 16px; margin: 0 0 4px; padding-right: 60px; }
.mono { font-family: var(--mono); }
.muted { color: var(--muted); }
.detail .spec h3 { font-size: 12px; color: var(--muted); text-transform: uppercase; margin: 16px 0 4px; }
.detail .spec p { margin: 0 0 8px; white-space: pre-wrap; }
.detail .spec ul { margin: 0 0 8px; padding-left: 18px; }
.section-title {
  font-size: 11px; text-transform: uppercase; letter-spacing: 0.08em;
  color: var(--muted); margin: 16px 0 8px; font-weight: 600;
}
.session { margin-bottom: 16px; }
.session-head {
  font-size: 11px; text-transform: uppercase; letter-spacing: 0.08em;
  color: var(--accent); font-weight: 600; margin-bottom: 8px;
}
.turn { border-left: 2px solid var(--border); padding: 0 0 8px 10px; margin-bottom: 10px; }
.prompt {
  background: var(--bg3); border: 1px solid var(--border); border-radius: 6px;
  padding: 6px 8px; margin-bottom: 8px; word-break: break-word;
}
.prompt p { margin: 0; }
.step { margin-bottom: 8px; }
.reasoning { color: var(--muted); font-size: 12px; margin-bottom: 6px; word-break: break-word; }
.reasoning p { margin: 0; }
.assistant { color: var(--fg); margin-bottom: 6px; word-break: break-word; }
.assistant p { margin: 0; }
.tool { border: 1px solid var(--border); border-radius: 6px; margin-bottom: 6px; overflow: hidden; }
.tool-head {
  display: flex; gap: 8px; align-items: center;
  background: var(--bg3); padding: 4px 8px; font-family: var(--mono); font-size: 12px;
}
.tool-name { color: var(--yellow); }
.tool-err { color: var(--red); font-size: 10px; text-transform: uppercase; }
.tool pre {
  margin: 0; padding: 6px 8px; overflow-x: auto;
  font-family: var(--mono); font-size: 11px; line-height: 1.45; max-height: 320px;
}
.tool .err { color: var(--red); padding: 6px 8px; white-space: pre-wrap; word-break: break-word; }
.tool-path { padding: 6px 8px; font-family: var(--mono); font-size: 11px; color: var(--accent); }
.tool-note { padding: 6px 8px; font-size: 11px; color: var(--muted); }

/* markdown + syntax highlighting */
.chroma { background: transparent; }
.prompt .chroma, .reasoning .chroma, .assistant .chroma, .tool-out .chroma {
  margin: 0; padding: 6px 8px; overflow-x: auto;
  font-family: var(--mono); font-size: 11px; line-height: 1.45;
}
.tool-out { padding: 0; }
.tool-out .chroma { max-height: 320px; }
.tool-diff { padding: 0; }
.tool-diff .chroma {
  margin: 0; padding: 6px 8px; overflow-x: auto;
  font-family: var(--mono); font-size: 11px; line-height: 1.45; max-height: 320px;
}
.chroma .c { color: #6272a4; }
.chroma .k, .chroma .kc, .chroma .kd, .chroma .kn, .chroma .kp, .chroma .kr, .chroma .kt { color: #ff79c6; }
.chroma .s, .chroma .s1, .chroma .s2, .chroma .sb, .chroma .sc { color: #f1fa8c; }
.chroma .m, .chroma .mf, .chroma .mi, .chroma .mo { color: #bd93f9; }
.chroma .n, .chroma .na, .chroma .nc, .chroma .nf, .chroma .nn { color: #8be9fd; }
.chroma .o, .chroma .p { color: #f8f8f2; }
.chroma .gd { color: #ff5555; }
.chroma .gi { color: #50fa7b; }
.chroma .gh { color: #f1fa8c; font-weight: bold; }
.chroma .gu { color: #6272a4; }
`
}

// shortID shortens a task id for display.
func shortID(s string) string { return shorten(s, 13) }

// shortHash shortens a commit hash for display.
func shortHash(s string) string { return shorten(s, 8) }

// displayStatus returns the live pipeline phase when a task is running,
// otherwise its stored status — so the badge and the phase line agree.
func displayStatus(status, phase string) string {
	if phase != "" {
		return phase
	}
	return status
}

// shorten truncates s to at most n runes.
func shorten(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
