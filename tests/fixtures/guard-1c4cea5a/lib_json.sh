#!/usr/bin/env sh
# Shared JSON field extraction for hooks.
# Source this file, then call: extract_json_field "$payload" "field_name"
#
# The field argument accepts a dotted path (e.g. "tool_input.file_path")
# to reach nested values. Top-level keys work without a dot.
#
# Uses jq when available for correct handling of escaped characters.
# Falls back to sed when jq is absent. The sed fallback reads a string
# value up to its closing quote, skipping JSON escapes (so \" inside the
# value does not end it), then turns \" back into " and \\ back into \.
# Other escapes (\n, \t, \uXXXX) stay as written: a multi-line value comes
# back on one line with literal \n. It expects the payload on one line
# (hooks pipe it through `tr '\n' ' '`) and only reads string values. It
# matches the leaf key name anywhere in the payload (the last occurrence
# wins), which works for the common case of unique key names, but users
# with ambiguous payloads should install jq.

extract_json_field() {
  _payload="$1"
  _field="$2"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$_payload" | jq -r ".${_field} // empty" 2>/dev/null
  else
    _leaf="${_field##*.}"
    printf '%s' "$_payload" \
      | sed -n -E "s/.*\"${_leaf}\"[[:space:]]*:[[:space:]]*\"(([^\"\\\\]|\\\\.)*)\".*/\\1/p" \
      | sed -E 's/\\(["\\])/\1/g'
  fi
}
