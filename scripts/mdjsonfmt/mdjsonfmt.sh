#!/usr/bin/env bash
# mdjsonfmt reformats ```json fenced code blocks inside Markdown files with jq
# (sorted-key-free, 2-space indent), leaving everything outside the fences
# untouched.
#
# Usage:
#   mdjsonfmt.sh [-c] FILE [FILE...]
#   mdjsonfmt.sh (-h | --help)
#
# Options:
#   -c, --check   Do not modify files. Report files whose JSON blocks would be
#                 rewritten and exit 1 if any exist. Useful in CI/lint.
#
# Exit codes:
#   0  all files are formatted (or were formatted successfully)
#   1  --check found at least one file needing formatting
#   2  usage error, unreadable file, malformed fence, or invalid JSON

set -u

if ! command -v jq >/dev/null 2>&1; then
	echo "mdjsonfmt: jq is required but not installed (https://jqlang.github.io/jq)" >&2
	exit 2
fi

usage() {
	sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'
}

CHECK=0
case "${1:-}" in
-c | --check)
	CHECK=1
	shift
	;;
-h | --help)
	usage
	exit 0
	;;
esac

if [ $# -lt 1 ]; then
	usage >&2
	exit 2
fi

TMPDIR_FMT="$(mktemp -d)" || exit 2
trap 'rm -rf "$TMPDIR_FMT"' EXIT

rc=0
need_fmt=0

for f in "$@"; do
	if [ ! -f "$f" ]; then
		echo "mdjsonfmt: $f: no such file" >&2
		rc=2
		continue
	fi

	out="$TMPDIR_FMT/out"
	block="$TMPDIR_FMT/block"
	jqerr="$TMPDIR_FMT/jqerr"
	: >"$out"

	inblock=0
	buf=""
	failed=0
	while IFS= read -r line || [ -n "$line" ]; do
		if [ "$inblock" -eq 1 ]; then
			case "$line" in
			'```')
				if [ -z "$buf" ]; then
					echo "mdjsonfmt: $f: empty json fence" >&2
					failed=1
					break
				fi
				if ! printf '%s' "$buf" | jq . >"$block" 2>"$jqerr"; then
					echo "mdjsonfmt: $f: invalid JSON in fenced block: $(tr -d '\n' <"$jqerr")" >&2
					failed=1
					break
				fi
				cat "$block" >>"$out"
				printf '%s\n' '```' >>"$out"
				inblock=0
				buf=""
				;;
			*)
				buf+="$line"$'\n'
				;;
			esac
		else
			case "$line" in
			'```json' | '```json '[[:space:]]*)
				printf '%s\n' "$line" >>"$out"
				inblock=1
				;;
			*)
				printf '%s\n' "$line" >>"$out"
				;;
			esac
		fi
	done <"$f"

	if [ "$failed" -eq 1 ]; then
		rc=2
		continue
	fi

	if [ "$inblock" -eq 1 ]; then
		echo "mdjsonfmt: $f: unterminated \`\`\`json fence" >&2
		rc=2
		continue
	fi

	if ! cmp -s "$f" "$out"; then
		if [ "$CHECK" -eq 1 ]; then
			echo "mdjsonfmt: $f: needs formatting"
			need_fmt=1
		else
			cp "$out" "$f"
			echo "mdjsonfmt: formatted $f"
		fi
	fi
done

if [ "$rc" -ne 0 ]; then
	exit "$rc"
fi
if [ "$need_fmt" -eq 1 ]; then
	exit 1
fi
exit 0
