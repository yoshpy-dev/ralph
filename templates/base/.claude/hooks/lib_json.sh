#!/usr/bin/env sh
# Shared JSON field extraction for hooks.
# Source this file, then call: extract_json_field "$payload" "field_name"
#
# Uses jq when available for correct handling of escaped characters.
# Falls back to sed (works for most payloads but fragile with \" in values).

extract_json_field() {
  _payload="$1"
  _field="$2"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$_payload" | jq -r ".[\"$_field\"] // empty" 2>/dev/null
  else
    printf '%s' "$_payload" | sed -n "s/.*\"${_field}\"[[:space:]]*:[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p"
  fi
}
