package upgrade

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// OwnedSettingsPaths lists the JSON paths inside .claude/settings.json that
// ralph owns and will merge. Anything outside these paths is preserved
// byte-for-byte (semantically) from the user's current file.
//
//   - "env": top-level object; ralph-shipped keys are kept in sync, user-added
//     keys are preserved.
//   - "permissions.allow", "permissions.deny": owned arrays, merged 3-way.
//   - "hooks": owned object; every event key under it (e.g. "PreToolUse") is an
//     owned array, merged 3-way.
var OwnedSettingsPaths = [...]string{
	"env",
	"permissions.allow",
	"permissions.deny",
	"hooks",
}

// SettingsMergeResult is the outcome of MergeOwnedSettings.
type SettingsMergeResult struct {
	// Content is the merged settings.json, 2-space indented with a trailing
	// newline, key order preserved (new keys appended at the end).
	Content []byte
	// Changed reports whether Content differs from the canonicalized form of
	// current. Callers use this to decide whether a write is needed.
	Changed bool
}

// MergeOwnedSettings performs a pure (no file I/O) 3-way merge of a
// .claude/settings.json file's ralph-owned paths (see OwnedSettingsPaths).
//
// current is the user's settings.json as it exists on disk; an empty or
// absent file is treated as "{}". oldOwned is the previously applied
// ralph-owned template (the state ralph last wrote); newOwned is the new
// ralph-owned template being upgraded to.
//
// Owned scalar/object values (env keys) are set to the new template value.
// Owned arrays (permissions.allow, permissions.deny, each hooks.<Event>
// array) are merged entry-by-entry: template entries are ensured present (in
// template order, deduplicated), entries that were ralph-owned in oldOwned
// but dropped from newOwned are removed, and entries in current that were
// never ralph-owned (i.e. absent from oldOwned) are preserved after the
// template entries, in their existing relative order. Keys outside
// OwnedSettingsPaths are never modified or removed.
//
// The result is deterministic: merging the same inputs twice yields
// identical bytes, and merging the output of a merge against the same
// templates again is a no-op.
func MergeOwnedSettings(current, oldOwned, newOwned []byte) (SettingsMergeResult, error) {
	curDoc, err := parseSettingsDoc(current)
	if err != nil {
		return SettingsMergeResult{}, fmt.Errorf("current settings.json: %w", err)
	}
	oldDoc, err := parseSettingsDoc(oldOwned)
	if err != nil {
		return SettingsMergeResult{}, fmt.Errorf("old owned template settings.json: %w", err)
	}
	newDoc, err := parseSettingsDoc(newOwned)
	if err != nil {
		return SettingsMergeResult{}, fmt.Errorf("new owned template settings.json: %w", err)
	}

	before := marshalOrdered(curDoc)

	mergeEnv(curDoc, oldDoc, newDoc)
	mergeOwnedArray(curDoc, oldDoc, newDoc, []string{"permissions", "allow"})
	mergeOwnedArray(curDoc, oldDoc, newDoc, []string{"permissions", "deny"})
	pruneEmptyChild(curDoc, "permissions")
	mergeHooks(curDoc, oldDoc, newDoc)

	after := marshalOrdered(curDoc)
	return SettingsMergeResult{Content: after, Changed: !bytes.Equal(before, after)}, nil
}

// mergeEnv syncs the top-level "env" object: keys present in the new
// template are set to the template value; keys that were ralph-owned in the
// old template but dropped from the new one are removed; every other key
// (never shipped by ralph) is left untouched.
func mergeEnv(merged, oldDoc, newDoc jsonValue) {
	oldEnv := getObjPath(oldDoc, "env")
	newEnv := getObjPath(newDoc, "env")
	if objEmpty(oldEnv) && objEmpty(newEnv) {
		return
	}

	envObj := ensureChildObj(merged.objVal, "env")

	if oldEnv != nil {
		for _, k := range oldEnv.keys {
			if newEnv == nil {
				envObj.delete(k)
				continue
			}
			if _, stillOwned := newEnv.vals[k]; !stillOwned {
				envObj.delete(k)
			}
		}
	}
	if newEnv != nil {
		for _, k := range newEnv.keys {
			envObj.set(k, newEnv.vals[k])
		}
	}
	if len(envObj.keys) == 0 {
		merged.objVal.delete("env")
	}
}

// mergeOwnedArray 3-way merges a single owned array path (e.g.
// permissions.allow) rooted at merged, using oldDoc/newDoc as the old/new
// owned templates. If both templates lack the path, nothing is touched
// (including not creating parent containers).
func mergeOwnedArray(merged, oldDoc, newDoc jsonValue, path []string) {
	oldArr := getArrayPath(oldDoc, path)
	newArr := getArrayPath(newDoc, path)
	if len(oldArr) == 0 && len(newArr) == 0 {
		return
	}
	curArr := getArrayPath(merged, path)

	result := merge3WayArray(curArr, oldArr, newArr)
	setOwnedArray(merged, path, result)
}

// mergeHooks syncs every owned event array under the top-level "hooks"
// object. An event key is considered owned only if it appears in the old or
// new owned template; event keys the user added on their own (never shipped
// by ralph) are left completely untouched.
func mergeHooks(merged, oldDoc, newDoc jsonValue) {
	oldHooks := getObjPath(oldDoc, "hooks")
	newHooks := getObjPath(newDoc, "hooks")
	if objEmpty(oldHooks) && objEmpty(newHooks) {
		return
	}

	curHooks := getObjPath(merged, "hooks")
	events := unionKeysOrdered(newHooks, oldHooks)

	for _, event := range events {
		curArr := arrValOrNil(curHooks, event)
		oldArr := arrValOrNil(oldHooks, event)
		newArr := arrValOrNil(newHooks, event)

		result := merge3WayArray(curArr, oldArr, newArr)
		setOwnedArray(merged, []string{"hooks", event}, result)
	}

	pruneEmptyChild(merged, "hooks")
}

// pruneEmptyChild removes key from merged's top-level object if it is
// present and is now an empty object, so owned-but-now-unused containers
// (e.g. "permissions" with no allow/deny left, "hooks" with no events left)
// don't linger as dangling "{}" entries.
func pruneEmptyChild(merged jsonValue, key string) {
	if child := getObjPath(merged, key); child != nil && len(child.keys) == 0 {
		merged.objVal.delete(key)
	}
}

// setOwnedArray writes result at path under merged, creating owned parent
// objects as needed, or removes the leaf key entirely when result is empty
// (owned arrays are not left behind as dangling "[]").
func setOwnedArray(merged jsonValue, path []string, result []jsonValue) {
	obj := merged.objVal
	for _, p := range path[:len(path)-1] {
		obj = ensureChildObj(obj, p)
	}
	leaf := path[len(path)-1]
	if len(result) == 0 {
		obj.delete(leaf)
		return
	}
	obj.set(leaf, jsonValue{kind: kindArray, arrVal: result})
}

// merge3WayArray computes the 3-way merged array for one owned array path:
// template entries first (new-template order, deduplicated), followed by
// entries from cur that were never ralph-owned (absent from oldOwned),
// preserving their relative order and deduplicated against everything
// already included.
//
// Known behavior: if a user-added entry is byte-for-byte identical to an
// entry that was ralph-owned in oldOwned, it is treated as that ralph entry
// (entry identity is structural, not provenance-tracked) — so if newOwned
// drops it, the user's duplicate is removed too.
func merge3WayArray(cur, oldOwned, newOwned []jsonValue) []jsonValue {
	var result []jsonValue
	appendUnique := func(e jsonValue) {
		for _, r := range result {
			if jsonEqual(r, e) {
				return
			}
		}
		result = append(result, e)
	}

	for _, e := range newOwned {
		appendUnique(e)
	}
	for _, e := range cur {
		if containsEntry(oldOwned, e) {
			continue
		}
		appendUnique(e)
	}
	return result
}

func containsEntry(list []jsonValue, e jsonValue) bool {
	for _, v := range list {
		if jsonEqual(v, e) {
			return true
		}
	}
	return false
}

// ---- ordered JSON value model ----
//
// encoding/json decodes objects into Go maps, which lose key order. Settings
// merge needs to preserve the user's existing top-level key order (append
// new keys at the end) and produce deterministic output, so this is a small
// order-preserving JSON value tree used only within this file.

type jsonKind int

const (
	kindNull jsonKind = iota
	kindBool
	kindNumber
	kindString
	kindArray
	kindObject
)

type jsonValue struct {
	kind   jsonKind
	boolV  bool
	numV   json.Number
	strV   string
	arrVal []jsonValue
	objVal *orderedObj
}

type orderedObj struct {
	keys []string
	vals map[string]jsonValue
}

func newOrderedObj() *orderedObj {
	return &orderedObj{vals: make(map[string]jsonValue)}
}

func (o *orderedObj) set(key string, val jsonValue) {
	if _, exists := o.vals[key]; !exists {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = val
}

func (o *orderedObj) delete(key string) {
	if _, exists := o.vals[key]; !exists {
		return
	}
	delete(o.vals, key)
	for i, k := range o.keys {
		if k == key {
			o.keys = append(o.keys[:i], o.keys[i+1:]...)
			break
		}
	}
}

// parseSettingsDoc decodes data into an ordered JSON object value. Empty (or
// whitespace-only) input is treated as an empty object. The root must be a
// JSON object.
func parseSettingsDoc(data []byte) (jsonValue, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return jsonValue{kind: kindObject, objVal: newOrderedObj()}, nil
	}

	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()
	val, err := decodeJSONValue(dec)
	if err != nil {
		return jsonValue{}, err
	}
	if dec.More() {
		return jsonValue{}, fmt.Errorf("trailing data after top-level JSON value")
	}
	if val.kind != kindObject {
		return jsonValue{}, fmt.Errorf("root must be a JSON object")
	}
	return val, nil
}

func decodeJSONValue(dec *json.Decoder) (jsonValue, error) {
	tok, err := dec.Token()
	if err != nil {
		return jsonValue{}, err
	}
	return decodeJSONValueFromToken(dec, tok)
}

func decodeJSONValueFromToken(dec *json.Decoder, tok json.Token) (jsonValue, error) {
	switch v := tok.(type) {
	case json.Delim:
		switch v {
		case '{':
			obj := newOrderedObj()
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return jsonValue{}, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return jsonValue{}, fmt.Errorf("expected object key, got %v", keyTok)
				}
				val, err := decodeJSONValue(dec)
				if err != nil {
					return jsonValue{}, err
				}
				obj.set(key, val)
			}
			if _, err := dec.Token(); err != nil { // consume '}'
				return jsonValue{}, err
			}
			return jsonValue{kind: kindObject, objVal: obj}, nil
		case '[':
			var arr []jsonValue
			for dec.More() {
				val, err := decodeJSONValue(dec)
				if err != nil {
					return jsonValue{}, err
				}
				arr = append(arr, val)
			}
			if _, err := dec.Token(); err != nil { // consume ']'
				return jsonValue{}, err
			}
			return jsonValue{kind: kindArray, arrVal: arr}, nil
		default:
			return jsonValue{}, fmt.Errorf("unexpected delimiter %v", v)
		}
	case bool:
		return jsonValue{kind: kindBool, boolV: v}, nil
	case json.Number:
		return jsonValue{kind: kindNumber, numV: v}, nil
	case string:
		return jsonValue{kind: kindString, strV: v}, nil
	case nil:
		return jsonValue{kind: kindNull}, nil
	default:
		return jsonValue{}, fmt.Errorf("unexpected token %v (%T)", tok, tok)
	}
}

func jsonEqual(a, b jsonValue) bool {
	if a.kind != b.kind {
		return false
	}
	switch a.kind {
	case kindNull:
		return true
	case kindBool:
		return a.boolV == b.boolV
	case kindNumber:
		return a.numV.String() == b.numV.String()
	case kindString:
		return a.strV == b.strV
	case kindArray:
		if len(a.arrVal) != len(b.arrVal) {
			return false
		}
		for i := range a.arrVal {
			if !jsonEqual(a.arrVal[i], b.arrVal[i]) {
				return false
			}
		}
		return true
	case kindObject:
		if a.objVal == nil || b.objVal == nil {
			return a.objVal == b.objVal
		}
		if len(a.objVal.keys) != len(b.objVal.keys) {
			return false
		}
		for _, k := range a.objVal.keys {
			bv, ok := b.objVal.vals[k]
			if !ok {
				return false
			}
			if !jsonEqual(a.objVal.vals[k], bv) {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// ---- path helpers ----

func getObjPath(root jsonValue, keys ...string) *orderedObj {
	cur := root
	for _, k := range keys {
		if cur.kind != kindObject || cur.objVal == nil {
			return nil
		}
		v, ok := cur.objVal.vals[k]
		if !ok {
			return nil
		}
		cur = v
	}
	if cur.kind != kindObject {
		return nil
	}
	return cur.objVal
}

func getArrayPath(root jsonValue, keys []string) []jsonValue {
	cur := root
	for i, k := range keys {
		if cur.kind != kindObject || cur.objVal == nil {
			return nil
		}
		v, ok := cur.objVal.vals[k]
		if !ok {
			return nil
		}
		if i == len(keys)-1 {
			if v.kind != kindArray {
				return nil
			}
			return v.arrVal
		}
		cur = v
	}
	return nil
}

func arrValOrNil(obj *orderedObj, key string) []jsonValue {
	if obj == nil {
		return nil
	}
	v, ok := obj.vals[key]
	if !ok || v.kind != kindArray {
		return nil
	}
	return v.arrVal
}

func objEmpty(obj *orderedObj) bool {
	return obj == nil || len(obj.keys) == 0
}

// ensureChildObj returns the object value at key under parent, creating (or
// replacing a non-object value at that key with) an empty object as needed.
// Replacing a non-object value only happens at an owned path, where ralph is
// authoritative over the container shape.
func ensureChildObj(parent *orderedObj, key string) *orderedObj {
	v, ok := parent.vals[key]
	if ok && v.kind == kindObject && v.objVal != nil {
		return v.objVal
	}
	child := newOrderedObj()
	parent.set(key, jsonValue{kind: kindObject, objVal: child})
	return child
}

// unionKeysOrdered returns the union of preferred's and secondary's keys,
// preferred's order first, then any secondary-only keys in secondary's
// order.
func unionKeysOrdered(preferred, secondary *orderedObj) []string {
	seen := make(map[string]bool)
	var out []string
	if preferred != nil {
		for _, k := range preferred.keys {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	if secondary != nil {
		for _, k := range secondary.keys {
			if !seen[k] {
				seen[k] = true
				out = append(out, k)
			}
		}
	}
	return out
}

// ---- serialization ----

// marshalOrdered renders v as 2-space indented JSON with a trailing newline,
// preserving object key order exactly as stored.
func marshalOrdered(v jsonValue) []byte {
	var buf bytes.Buffer
	writeJSONValue(&buf, v, 0)
	buf.WriteByte('\n')
	return buf.Bytes()
}

func writeJSONValue(buf *bytes.Buffer, v jsonValue, indent int) {
	switch v.kind {
	case kindNull:
		buf.WriteString("null")
	case kindBool:
		if v.boolV {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case kindNumber:
		buf.WriteString(v.numV.String())
	case kindString:
		buf.WriteString(marshalJSONString(v.strV))
	case kindArray:
		writeJSONArray(buf, v.arrVal, indent)
	case kindObject:
		writeJSONObject(buf, v.objVal, indent)
	}
}

func writeJSONObject(buf *bytes.Buffer, obj *orderedObj, indent int) {
	if obj == nil || len(obj.keys) == 0 {
		buf.WriteString("{}")
		return
	}
	buf.WriteString("{\n")
	childIndent := indent + 2
	pad := strings.Repeat(" ", childIndent)
	for i, k := range obj.keys {
		buf.WriteString(pad)
		buf.WriteString(marshalJSONString(k))
		buf.WriteString(": ")
		writeJSONValue(buf, obj.vals[k], childIndent)
		if i < len(obj.keys)-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.WriteString(strings.Repeat(" ", indent))
	buf.WriteByte('}')
}

func writeJSONArray(buf *bytes.Buffer, arr []jsonValue, indent int) {
	if len(arr) == 0 {
		buf.WriteString("[]")
		return
	}
	buf.WriteString("[\n")
	childIndent := indent + 2
	pad := strings.Repeat(" ", childIndent)
	for i, item := range arr {
		buf.WriteString(pad)
		writeJSONValue(buf, item, childIndent)
		if i < len(arr)-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.WriteString(strings.Repeat(" ", indent))
	buf.WriteByte(']')
}

// marshalJSONString renders s as a quoted JSON string without HTML escaping,
// so common settings.json content (e.g. "Bash(a && b)" permission patterns)
// stays readable instead of turning into & escapes.
func marshalJSONString(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		// s is a valid Go string; Encode of a string cannot fail.
		panic(err)
	}
	return strings.TrimSuffix(buf.String(), "\n")
}
