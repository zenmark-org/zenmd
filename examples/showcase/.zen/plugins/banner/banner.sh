#!/bin/sh
# .zen/plugins/banner/banner.sh — renders :::banner, :::alert and :::notice containers
# (and the same names written the MyST way, see plugins/dialects.md)
esc() { sed -e 's/&/\&amp;/g' -e 's/</\&lt;/g' -e 's/>/\&gt;/g'; }
case "$ZEN_NAME" in
  alert) kind=alert title=Alert ;;
  notice) kind=banner title=Notice ;;
  *) kind=banner title=Banner ;;
esac
title="${ZEN_ATTR_TITLE:-$title}"
printf '<div class="%s"><strong>' "$kind"
printf '%s' "$title" | esc
printf '</strong>\n'
esc                                  # the body arrives on stdin
printf '</div>'
