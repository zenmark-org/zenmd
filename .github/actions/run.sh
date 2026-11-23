#!/bin/sh
# Runs zenmd for the docs CI action (specs/packaging/docs-ci): the optional
# check, then `zenmd export site`, with the warnings as GitHub annotations and
# the outputs written to $GITHUB_OUTPUT.
#
# zenmd prints one warning per stderr line: "<page>: <text>" (the page's path
# in the folder), or "site: <text>" for the whole site; a text may start with
# "line N: ". Annotations carry that text only.
#
# Environment: ZM_FOLDER, ZM_OUTPUT, ZM_HOSTED, ZM_DRAFTS, ZM_SCHEME, ZM_CLEAN,
# ZM_PRINT, ZM_CHECK, GITHUB_OUTPUT.
set -eu

folder="${ZM_FOLDER:-.}"
output="${ZM_OUTPUT:-_site}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# annotate <file of warning lines>: one ::warning:: per line; prints the count
# to $tmp/count.
annotate() {
  n=0
  while IFS= read -r line || [ -n "$line" ]; do
    [ -n "$line" ] || continue
    n=$((n + 1))
    file="" lno="" msg="$line"
    case "$line" in
      *": "*)
        head="${line%%: *}"
        msg="${line#*: }"
        if [ "$head" != site ] && [ "$head" != zenmd ]; then
          file="$head"
          case "$msg" in
            "line "[0-9]*": "*)
              rest="${msg#line }"
              lno="${rest%%:*}"
              msg="${rest#*: }"
              ;;
          esac
        else
          msg="$line"
        fi
        ;;
    esac
    if [ -n "$file" ]; then
      case "$folder" in
        .|./|"") ;;
        *) file="${folder%/}/${file#./}" ;;
      esac
      file="$(printf '%s' "$file" | sed -e 's/%/%25/g' -e 's/,/%2C/g' -e 's/:/%3A/g')"
    fi
    msg="$(printf '%s' "$msg" | sed -e 's/%/%25/g' -e 's/\r/%0D/g')"
    props=""
    [ -z "$file" ] || props="file=$file,"
    [ -z "$lno" ] || props="${props}line=$lno,"
    printf '::warning %stitle=zenmd::%s\n' "$props" "$msg"
  done < "$1"
  echo "$n" > "$tmp/count"
}

if [ "${ZM_CHECK:-false}" = true ]; then
  if ! zenmd help 2>/dev/null | grep -q 'zenmd check'; then
    echo "check: true needs a zenmd with the check command (0.8.0 or later)" >&2
    exit 1
  fi
  rc=0
  zenmd check "$folder" >"$tmp/check.out" 2>"$tmp/check.err" || rc=$?
  cat "$tmp/check.out"
  if [ "$rc" -ne 0 ]; then
    cat "$tmp/check.err" >&2
    exit "$rc"
  fi
  annotate "$tmp/check.err"
fi

set -- export site "$folder" -o "$output"
if [ "${ZM_HOSTED:-true}" = true ]; then set -- "$@" --hosted; else set -- "$@" --portable; fi
[ "${ZM_DRAFTS:-false}" != true ] || set -- "$@" --drafts
[ -z "${ZM_SCHEME:-}" ] || set -- "$@" --scheme "$ZM_SCHEME"
[ "${ZM_CLEAN:-true}" != true ] || set -- "$@" --clean
[ "${ZM_PRINT:-true}" = true ] || set -- "$@" --no-print

rc=0
zenmd "$@" >"$tmp/out" 2>"$tmp/err" || rc=$?
cat "$tmp/out"
if [ "$rc" -ne 0 ]; then
  cat "$tmp/err" >&2
  exit "$rc"
fi
annotate "$tmp/err"
warnings="$(cat "$tmp/count")"
pages="$(sed -n 's/^Built \([0-9][0-9]*\) pages\{0,1\} .*/\1/p' "$tmp/out" | head -n 1)"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    echo "output=$output"
    echo "pages=${pages:-0}"
    echo "warnings=$warnings"
  } >> "$GITHUB_OUTPUT"
fi
