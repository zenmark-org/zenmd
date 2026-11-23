#!/bin/sh
# Runs zenmd for the docs CI action (specs/packaging/docs-ci): the optional
# check, then `zenmd export site`, with the warnings as GitHub annotations and
# the outputs written to $GITHUB_OUTPUT.
#
# zenmd prints one warning per stderr line: "<page>: <text>" (the page's path
# in the folder), or "site: <text>" for the whole site; a text may start with
# "line N: ". `zenmd check` prints one problem per line on stdout:
# "<page>:<line>: <severity>: <message>" (error, warning or suggestion), which
# becomes ::error, ::warning or ::notice with line=. Annotations carry that
# text only.
#
# Environment: ZM_FOLDER, ZM_OUTPUT, ZM_HOSTED, ZM_DRAFTS, ZM_SCHEME, ZM_CLEAN,
# ZM_PRINT, ZM_CHECK, GITHUB_OUTPUT.
set -eu

folder="${ZM_FOLDER:-.}"
output="${ZM_OUTPUT:-_site}"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

# esc_prop and esc_msg escape a property value and a message for a workflow command.
esc_prop() { printf '%s' "$1" | sed -e 's/%/%25/g' -e 's/,/%2C/g' -e 's/:/%3A/g'; }
esc_msg() { printf '%s' "$1" | sed -e 's/%/%25/g' -e 's/\r/%0D/g'; }

# with_folder <page>: the page's path from the repository root.
with_folder() {
  case "$folder" in
    .|./|"") printf '%s' "$1" ;;
    *) printf '%s/%s' "${folder%/}" "${1#./}" ;;
  esac
}

# annotate <file> <mode>: one annotation per line; prints the count to
# $tmp/count. A check-form line ("<page>:<line>: <severity>: <message>") maps
# error, warning and suggestion to ::error, ::warning and ::notice. In mode
# check, any other line is passed through as it is; in mode export it is a
# warning of the "<page>: <text>" form.
annotate() {
  n=0
  while IFS= read -r line || [ -n "$line" ]; do
    [ -n "$line" ] || continue
    if printf '%s\n' "$line" | grep -Eq '^.+:[0-9]+: (error|warning|suggestion): '; then
      n=$((n + 1))
      sev="$(printf '%s\n' "$line" | sed -E 's/^.+:[0-9]+: (error|warning|suggestion): .*$/\1/')"
      lno="$(printf '%s\n' "$line" | sed -E 's/^.+:([0-9]+): (error|warning|suggestion): .*$/\1/')"
      pg="$(printf '%s\n' "$line" | sed -E 's/^(.+):[0-9]+: (error|warning|suggestion): .*$/\1/')"
      msg="$(printf '%s\n' "$line" | sed -E 's/^.+:[0-9]+: (error|warning|suggestion): (.*)$/\2/')"
      case "$sev" in error) cmd=error ;; warning) cmd=warning ;; *) cmd=notice ;; esac
      printf '::%s file=%s,line=%s,title=zenmd::%s\n' "$cmd" "$(esc_prop "$(with_folder "$pg")")" "$lno" "$(esc_msg "$msg")"
      continue
    fi
    if [ "$2" = check ]; then
      printf '%s\n' "$line"
      continue
    fi
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
      file="$(esc_prop "$(with_folder "$file")")"
    fi
    msg="$(esc_msg "$msg")"
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
  zenmd check "$folder" >"$tmp/check.out" 2>&1 || rc=$?
  annotate "$tmp/check.out" check
  [ "$rc" -eq 0 ] || exit "$rc"
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
annotate "$tmp/err" export
warnings="$(cat "$tmp/count")"
pages="$(sed -n 's/^Built \([0-9][0-9]*\) pages\{0,1\} .*/\1/p' "$tmp/out" | head -n 1)"

if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    echo "output=$output"
    echo "pages=${pages:-0}"
    echo "warnings=$warnings"
  } >> "$GITHUB_OUTPUT"
fi
