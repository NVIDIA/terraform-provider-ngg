// Copyright 2025 NVIDIA Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

var debug = true

func WriteMsg(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Print(msg)
	//UpdateSessionLogging()
	//if sessionLog != nil {
	//	if _, err := sessionLog.WriteString(msg); err != nil {
	//		fmt.Printf("Failed to write to session log! Error: %v\n", err)
	//	}
	//}
}

func WriteStringToFile(filename string, data string, label string, printErrors bool, useTemp bool) bool {
	var theFile *os.File = nil
	var err error = nil
	valid := true

	if useTemp {
		theFile, err = ioutil.TempFile("", "temp*")
	} else {
		theFile, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY, 0644)
	}
	if err != nil {
		valid = false
		if printErrors {
			fmt.Printf("Failed to open %s file (%s)! Error: %v\n", label, filename, err)
		}
		return false
	}
	if _, err = theFile.WriteString(data); err != nil {
		valid = false
		if printErrors {
			fmt.Printf("Failed to write to %s file (%s)! Error: %v\n", label, filename, err)
		}
	}
	theFile.Close()
	if useTemp {
		if valid {
			os.Rename(theFile.Name(), filename)
		} else {
			os.Remove(theFile.Name())
		}
	}
	return true
}

func ReadStringFromFile(filename string, label string, printErrors bool) (string, bool) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if printErrors {
			fmt.Printf("Error while reading %s file (%s)! Error: %v\n", label, filename, err)
		}
		return "", false
	}
	return string(data), true
}

func ReadJsonFromFile(filename string) map[string]interface{} {
	objects := map[string]interface{}{}
	jsonStr, ok := ReadStringFromFile(filename, "Json Data", true)
	if !ok {
		return objects
	}
	// Parsing/Unmarshalling JSON encoding/json
	err := json.Unmarshal([]byte(jsonStr), &objects)
	if err != nil {
		if debug {
			//logging.WriteMsgColor(Red, "WARNING: Failed to parse JSON from parseSymbolTable().\n")
			WriteMsg("WARNING: Failed to parse JSON from parseSymbolTable().\n")
		}
		return nil
	}
	return objects
}

func WriteJsonToFile(filename string, js map[string]interface{}) bool {
	bs, err := json.Marshal(js)
	if err != nil {
		//logging.WriteMsg("unable to marshal data to json: %v\n", err)
		WriteMsg("unable to marshal data to json: %v\n", err)
		return false
	}
	return WriteStringToFile(filename, string(bs), "Json value", true, true)
}

// Convert a dotted-key to []string
func ToKeyPath(path string) []string {
	return strings.Split(strings.TrimSpace(path), ".")
}

// function to convert comma-separated dotted-key to [][]string
func ToKeyPathArray(paths string) [][]string {
	result := [][]string{}
	for _, path := range strings.Split(paths, ",") {
		result = append(result, ToKeyPath(path))
	}
	return result
}

// Returns a sorted list of all top-level keys that start with the given prefix.
func GetPrefixedKeys(objects map[string]interface{}, prefix string) []string {
	keysOut := []string{}
	for key, _ := range objects {
		if strings.HasPrefix(key, prefix) {
			keysOut = append(keysOut, key)
		}
	}
	sort.Strings(keysOut)
	return keysOut
}

func ParseIndexSpec(spec string) int {
	l := len(spec)
	if !strings.HasPrefix(spec, "[") {
		return -1
	}
	if !strings.HasSuffix(spec, "]") {
		return -1
	}
	subStr := spec[1 : l-1]
	if len(subStr) == 0 {
		return -2
	}
	parsed := CastToInt(subStr)
	if parsed < math.MinInt || parsed > math.MaxInt {
		return -1 // Out of bounds for int
	}
	return int(parsed)
}

// Returns the object at js[path[0]][path[1]][...],true if it exists or nil,false
func GetNestedValue(js interface{}, path []string) (interface{}, bool) {
	// TODO handle wildcards
	// TODO handle slice notation
	var cur = js
	var ok bool
	for _, p := range path {
		switch cur.(type) {
		case map[string]interface{}:
			cur, ok = cur.(map[string]interface{})[p]
			if !ok {
				return nil, false
			}
		case []interface{}:
			idx := ParseIndexSpec(p)
			arr := cur.([]interface{})
			if idx < 0 || idx >= len(arr) {
				return nil, false
			}
			cur = arr[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

// Returns the object at js[path[0]][path[1]][...] if it exists or 'dfault'
func GetNestedValueOrDefault(js interface{}, path []string, dfault interface{}) interface{} {
	val, ok := GetNestedValue(js, path)
	if !ok {
		return dfault
	}
	return val
}

func SetNestedValue(js interface{}, path []string, val interface{}) interface{} {
	l := len(path)
	if l == 0 {
		return nil
	}
	lastKey := path[l-1]
	parent := js
	if l > 1 {
		parent, _ = GetNestedValue(js, path[0:l-1])
	}
	if parent == nil {
		return nil
	}
	switch parent.(type) {
	case map[string]interface{}:
		cur, ok := parent.(map[string]interface{})
		if !ok {
			return nil
		}
		cur[lastKey] = val
		return cur
	default:
		return nil
	}
}

func IsMapLeaf(key string, m map[string]interface{}) bool {
	switch m[key].(type) {
	case map[string]interface{}:
		return false
	case []interface{}:
		return false
	}
	return true
}

// NOTE: prefix should be empty or include a trailing dot '.'
func GetFlattenedKeys(js interface{}, prefix string, excludes map[string]bool) []string {
	keys := []string{}
	switch js.(type) {
	case map[string]interface{}:
		cur, ok := js.(map[string]interface{})
		if ok {
			for k, _ := range cur {
				subKey := prefix + k
				if !excludes[subKey] {
					if IsMapLeaf(k, cur) {
						keys = append(keys, subKey)
					} else {
						subKeys := GetFlattenedKeys(cur[k], subKey+".", excludes)
						keys = append(keys, subKeys...)
					}
				}
			}
		}
	}
	return keys
}

// Returns the array at js[path[0]][path[1]][...] if it exists or 'dfault'
func GetSimpleArrayNestedValue(js interface{}, path []string, pad bool) []interface{} {
	var result = []interface{}{}
	jsArray, ok := js.([]interface{})
	if ok {
		for _, c := range jsArray {
			val, ok := GetNestedValue(c, path)
			if ok || pad {
				result = append(result, val)
			}
		}
	} else {
		return nil
	}
	return result
}

// Returns a flattened version of 'ar'.
// Non-array sub-elements are kept, in order.
// Array sub-elements are taken out and placed in-order in the result.
// e.g.  [1, [2,3], 4, [5, [6,7], 8]  -> [1, 2, 3, 4, 5, [6, 7], 8]
func FlattenArray(ar []interface{}) []interface{} {
	result := []interface{}{}
	for _, v := range ar {
		va, isArray := v.([]interface{})
		if !isArray {
			result = append(result, v)
		} else {
			for _, sv := range va {
				result = append(result, sv)
			}
		}
	}
	return result
}

// Append 'val' to an array in map[key], creating an empty array if it doesn't exist.
func ExtendMapArray(js map[string]interface{}, key string, val interface{}) {
	sub, ok := js[key]
	if !ok {
		// TODO check type? but then what, replace with empty array?
		sub = []interface{}{}
	}
	subArray, ok := sub.([]interface{})
	if ok {
		js[key] = append(subArray, val)
	}
}

// Takes an array of key/value objects, and merges them into a map
func MergeKeyValueArrayToMap(theMap map[string]interface{}, array []interface{}, keyPath []string, valPath []string, overwrite bool) {
	for _, field := range array {
		curMap, isMap := field.(map[string]interface{})
		if isMap {
			key := GetNestedValueOrDefault(curMap, keyPath, nil)
			val := GetNestedValueOrDefault(curMap, valPath, nil)
			keyStr, isKeyStr := key.(string)
			if key != nil && val != nil && isKeyStr {
				// TODO covert non-alphanum to underscore or somesuch
				// Note: dots in 'key' cause nesting.
				AddNestedValue(theMap, ToKeyPath(keyStr), val, overwrite)
			}
		}
	}
}

// Converts an array to a map-of-arrays keyed on 'path'
func PartitionArrayBySubKey(js interface{}, path []string) map[string]interface{} {
	var result = map[string]interface{}{}
	jsArray, ok := js.([]interface{})
	if ok {
		for _, c := range jsArray {
			val, ok := GetNestedValue(c, path)
			if ok {
				strVal, ok := val.(string)
				if ok {
					ExtendMapArray(result, strVal, c)
				}
			}
		}
	} else {
		return nil
	}
	return result
}

// Inserts 'val' at js[path[0]][path[1]][...], creating empty intermediate maps as required
// Fails if any intermediate objects are not a string-map.
// Fails if the last element exists and 'overwrite' is false.
// TODO allow merging final object
// TODO better specification of overwrite at various levels
func AddNestedValue(js map[string]interface{}, path []string, val interface{}, overwrite bool) bool {
	var cur interface{} = js
	var plen = len(path)
	for i, p := range path {
		curMap, ok := cur.(map[string]interface{})
		if !ok {
			return false
		}
		cur, ok = curMap[p]
		if !ok {
			if i == plen-1 {
				curMap[p] = val
			} else {
				curMap[p] = map[string]interface{}{}
				cur = curMap[p]
			}
		} else {
			if i == plen-1 {
				if overwrite {
					curMap[p] = val
				} else {
					return false
				}
			}
		}
	}
	return false
}

// Removes the field at js[path[0]][path[1]][...]
// Returns true if the field was found and deleted.
func RemoveObjectNestedValue(js map[string]interface{}, path []string) bool {
	var cur interface{} = js
	var plen = len(path)
	var ok bool
	for i, p := range path {
		curMap, isMap := cur.(map[string]interface{})
		if !isMap {
			return false
		}
		cur, ok = curMap[p]
		if !ok {
			return false
		}
		if i == plen-1 {
			delete(curMap, p)
			return true
		}
	}
	return false
}

// Removes the fields (from an object) at js[path[N][0]][path[N][1]][...]
// Returns true if any field was found and deleted.
func RemoveObjectNestedValues(js map[string]interface{}, paths [][]string) bool {
	result := false
	for _, path := range paths {
		did := RemoveObjectNestedValue(js, path)
		result = result || did
	}
	return result
}

// Removes the fields (from each object in an array) at js[path[N][0]][path[N][1]][...]
// Returns true if any field was found and deleted.
func RemoveArrayNestedValues(js []interface{}, paths [][]string) bool {
	result := false
	for _, cur := range js {
		curMap, ok := cur.(map[string]interface{})
		if ok {
			did := RemoveObjectNestedValues(curMap, paths)
			result = result || did
		}
	}
	return result
}

// Returns an object with fields js[path[N][0]][path[N][1]][...] renamed to result[names[N][0]][names[N][1]][...]
// Missing source (js) fields are ignored
func GetObjectNestedValues(js interface{}, paths [][]string, names [][]string) map[string]interface{} {
	var result = map[string]interface{}{}
	for i, path := range paths {
		cur, ok := GetNestedValue(js, path)
		if ok {
			AddNestedValue(result, names[i], cur, true)
		}
	}
	return result
}

// Iterates over an array of objects extracting any fields at paths[n] from each and renaming them to names[N].
func GetObjectArrayNestedValues(js interface{}, paths [][]string, names [][]string) []interface{} {
	arr := js.([]interface{})
	result := []interface{}{}
	for _, cur := range arr {
		curObj := GetObjectNestedValues(cur, paths, names)
		result = append(result, curObj)
	}
	return result
}

// Re-orders objects in dest by matching the exemplar-array to dest[dest_key]
// Missing dest entries are padded with nil.
// Extra entries are ignored or appended based on 'appendExtra'.
func OrderObjectArray(exemplar []interface{}, dest []interface{}, dest_key []string, force_string bool, appendExtra bool) []interface{} {
	result := []interface{}{}
	usedMap := map[interface{}]bool{}
	destMap := map[interface{}]int{}
	for i, cur := range dest {
		orderKey, ok := GetNestedValue(cur, dest_key)
		if ok {
			if force_string {
				destMap[CastToString(orderKey)] = i
			} else {
				destMap[orderKey] = i
			}
		}
	}
	for _, cur := range exemplar {
		usedMap[cur] = true
		idx, ok := destMap[cur]
		if ok {
			result = append(result, dest[idx])
		} else {
			result = append(result, nil)
		}
	}
	if appendExtra {
		// Append entries that either don't have a 'dest_key' entry, or one that isn't in 'exemplar'
		// (e.g. a raw metric query that doesn't include host/pod/container)
		for _, cur := range dest {
			orderKey, ok := GetNestedValue(cur, dest_key)
			if !ok {
				// no ordering key
				result = append(result, cur)
			} else {
				if force_string {
					orderKey = CastToString(orderKey)
				}
				_, used := usedMap[orderKey]
				if !used {
					// not in exemplar
					result = append(result, cur)
				}
			}
		}
	}
	return result
}

// sort values that might be numbers (float64 for go JSON) or strings
func VagueLess(a interface{}, b interface{}) bool {
	an, aNum := a.(float64)
	bn, bNum := b.(float64)
	// numeric sort
	if aNum && bNum {
		return an < bn
	}
	// lexicographic sort
	as, aStr := a.(string)
	bs, bStr := b.(string)
	if aStr && bStr {
		return as < bs
	}
	// number first (arbitrary, could panic instead)
	return aNum
}

func MapToSortedDedupedArray(elements map[interface{}]bool) []interface{} {
	// back to (unsorted) array
	result := []interface{}{}
	for k, _ := range elements {
		result = append(result, k)
	}

	// sort based on type
	sort.Slice(result, func(i, j int) bool { return VagueLess(result[i], result[j]) })
	return result
}

func SortAndDedupArray(arr []interface{}) []interface{} {
	elementMap := map[interface{}]bool{}
	for _, val := range arr {
		// prevent duplicates
		elementMap[val] = true
	}
	return MapToSortedDedupedArray(elementMap)
}

// This is used to extract arrays (e.g. timestamps) from a set of top-level objects (e.g. metrics).
// The returned data (an array) is each of the sub-arrays combined, with values de-duped and sorted.
func ExtractAlignmentArray(js interface{}, align_key []string) []interface{} {
	elementMap := map[interface{}]bool{}
	jsMap, isMap := js.(map[string]interface{})
	if !isMap {
		return []interface{}{}
	}
	// pull out the arrays of alignment data (e.g. timestamps)
	for _, val := range jsMap {
		subArray, isArray := val.([]interface{})
		if !isArray {
			continue
		}
		for _, v := range subArray {
			cur := GetNestedValueOrDefault(v, align_key, nil)
			if cur == nil {
				continue
			}
			curAr, isArray := cur.([]interface{})
			if !isArray {
				continue
			}
			for _, av := range curAr {
				// prevent duplicates
				elementMap[av] = true
			}
		}
	}
	return MapToSortedDedupedArray(elementMap)
}

func AlignIndexedSubArrays(index int, js interface{}, align []interface{}, align_key []string, value_key []string) {
	jsMap, isMap := js.(map[string]interface{})
	if !isMap {
		return
	}
	// process the arrays of values by alignment data
	for _, val := range jsMap {
		subArray, isArray := val.([]interface{})
		if !isArray || len(subArray) <= index {
			continue
		}
		v := subArray[index]
		vMap, isMap := v.(map[string]interface{})
		if !isMap {
			continue
		}
		cur := GetNestedValueOrDefault(vMap, align_key, nil)
		if cur == nil {
			continue
		}
		curAr, isArray := cur.([]interface{})
		if !isArray {
			continue
		}

		elementMap := map[interface{}]int{}
		for i, av := range curAr {
			// lookup table of align_key to index
			elementMap[av] = i
		}

		valIn := GetNestedValueOrDefault(vMap, value_key, nil)
		arrayIn, isArray := valIn.([]interface{})
		if !isArray {
			continue
		}
		arrayOut := []interface{}{}
		for _, al := range align {
			idx, found := elementMap[al]
			if found {
				arrayOut = append(arrayOut, arrayIn[idx])
			} else {
				arrayOut = append(arrayOut, nil)
			}
		}
		AddNestedValue(vMap, align_key, align, true)
		AddNestedValue(vMap, value_key, arrayOut, true)
	}
}

// Take an array of values (e.g. global timestamps) 'align',
// assuming there's an array of similar values (e.g. local timestamps) under js[align_key],
// it null-pads an array under js[value_key] so that the values line up with the global array.
// Modifies the object in-place.
func AlignSubArrays(js interface{}, align []interface{}, align_key []string, value_key []string) {
	jsMap, isMap := js.(map[string]interface{})
	if !isMap {
		return
	}
	// process the arrays of values by alignment data
	for _, val := range jsMap {
		subArray, isArray := val.([]interface{})
		if !isArray {
			continue
		}
		for _, v := range subArray {
			vMap, isMap := v.(map[string]interface{})
			if !isMap {
				continue
			}
			cur := GetNestedValueOrDefault(vMap, align_key, nil)
			if cur == nil {
				continue
			}
			curAr, isArray := cur.([]interface{})
			if !isArray {
				continue
			}

			elementMap := map[interface{}]int{}
			for i, av := range curAr {
				// lookup table of align_key to index
				elementMap[av] = i
			}

			valIn := GetNestedValueOrDefault(vMap, value_key, nil)
			arrayIn, isArray := valIn.([]interface{})
			if !isArray {
				continue
			}
			arrayOut := []interface{}{}
			for _, al := range align {
				idx, found := elementMap[al]
				if found {
					arrayOut = append(arrayOut, arrayIn[idx])
				} else {
					arrayOut = append(arrayOut, nil)
				}
			}
			AddNestedValue(vMap, align_key, align, true)
			AddNestedValue(vMap, value_key, arrayOut, true)
		}
	}
}

// try to convert objects to their boolean representation (for string, bool, and numeric)
// returns <cast-value>,true for valid types
// returns false,false for nonsensical types/values (object, array, arbitrary strings)
func CastToBoolMaybe(val interface{}) (bool, bool) {
	switch val.(type) {
	case bool:
		return val.(bool), true
	case string:
		vs := strings.ToLower(val.(string))
		if vs == "true" || vs == "on" || vs == "1" {
			return true, true
		} else if vs == "false" || vs == "off" || vs == "0" {
			return false, true
		}
		return false, false
	case int:
		return val.(int) != 0, true
	case int64:
		return val.(int64) != 0, true
	case float32:
		return val.(float32) != 0, true
	case float64:
		return val.(float64) != 0, true
	default:
		return false, false
	}
}

// convert objects to their string representation (for objects (string-map), arrays, string, bool, and numbers)
func CastToString(val interface{}) string {
	// number, bool, null, object, array, string
	switch val.(type) {
	case map[string]interface{}:
		jstr, err := json.MarshalIndent(val, "", "  ")
		if err != nil {
			//logging.WriteMsg("CastToString(map) Marshal error: %v\n", err)
			WriteMsg("CastToString(map) Marshal error: %v\n", err)
			return ""
		}
		return string(jstr)
	case []interface{}:
		jstr, err := json.MarshalIndent(val, "", "  ")
		if err != nil {
			//logging.WriteMsg("CastToString(array) Marshal error: %v\n", err)
			WriteMsg("CastToString(array) Marshal error: %v\n", err)
			return ""
		}
		return string(jstr)
	case string:
		return val.(string)
	case bool:
		if val.(bool) {
			return "true"
		} else {
			return "false"
		}
	case int:
		return strconv.FormatInt(int64(val.(int)), 10)
	case int64:
		return strconv.FormatInt(val.(int64), 10)
	case float32:
		return strconv.FormatFloat(float64(val.(float32)), 'f', -1, 64)
	case float64:
		return strconv.FormatFloat(val.(float64), 'f', -1, 64) // value, format, precision, bits
	default:
		if val == nil {
			return "null"
		}
		return ""
	}
}

// convert objects to their numeric representation (for string, bool, and numeric)
// returns 0 for nonsensical types/values (object, array)
func CastToNumber(val interface{}) float64 {
	// from string, bool, number
	switch val.(type) {
	// XXX array to length?
	// XXX object to ???
	case string:
		v, err := strconv.ParseFloat(val.(string), 64)
		if err != nil {
			return 0
		}
		return v
	case bool:
		if val.(bool) {
			return 1
		} else {
			return 0
		}
	case int:
		return float64(val.(int))
	case int64:
		return float64(val.(int64))
	case float32:
		return float64(val.(float32))
	case float64:
		return val.(float64)
	default:
		return 0
	}
}

// convert objects to their numeric representation (for string, bool, and numeric)
// returns 0 for nonsensical types/values (object, array)
func CastToInt(val interface{}) int64 {
	// from string, bool, number
	switch val.(type) {
	// XXX array to length?
	// XXX object to ???
	case string:
		v, err := strconv.ParseInt(val.(string), 10, 64)
		if err != nil {
			return 0
		}
		return v
	case bool:
		if val.(bool) {
			return 1
		} else {
			return 0
		}
	case int:
		return int64(val.(int))
	case int64:
		return val.(int64)
	case float32:
		return int64(val.(float32))
	case float64:
		return int64(val.(float64))
	default:
		return 0
	}
}

// convert objects to their boolean representation (for string, bool, and numeric)
// returns false for nonsensical types/values (object, array)
func CastToBool(val interface{}) bool {
	// from string, bool, number, null
	switch val.(type) {
	// XXX array to ???
	// XXX object to ???
	case string:
		if strings.ToLower(val.(string)) == "true" {
			return true
		}
		f, err := strconv.ParseFloat(val.(string), 64)
		if err == nil && f != 0.0 {
			return true
		}
		return false
	case bool:
		return val.(bool)
	case int:
		return val.(int) != 0
	case int64:
		return val.(int64) != 0
	case float32:
		return val.(float32) != 0
	case float64:
		return val.(float64) != 0
	default:
		return false
	}
}

// convert objects to their boolean representation (for string, object)
// returns false for nonsensical types/values (array, numeric, bool)
func CastToObject(val interface{}) interface{} {
	// from string
	switch val.(type) {
	case map[string]interface{}:
		return val
	case string:
		var objects interface{}
		err := json.Unmarshal([]byte(val.(string)), &objects)
		if err != nil {
			//logging.WriteMsg("CastToObject(string) Unmarshal error: %v\n", err)
			WriteMsg("CastToObject(string) Unmarshal error: %v\n", err)
			return nil
		}
		return objects
	default:
		return nil
	}
}

func CastToArray(val interface{}) []interface{} {
	switch val.(type) {
	case []interface{}:
		return val.([]interface{})
	case string:
		var objects interface{}
		err := json.Unmarshal([]byte("{ \"arr\": "+val.(string)+"}"), &objects)
		if err != nil {
			//logging.WriteMsg("CastToObject(string) Unmarshal error: %v\n", err)
			WriteMsg("CastToObject(string) Unmarshal error: %v\n", err)
			return nil
		}
		arr, _ := GetNestedValueOrDefault(objects, ToKeyPath("arr"), nil).([]interface{})
		return arr
	}

	kind := reflect.TypeOf(val).Kind()
	if kind == reflect.Slice {
		// It's something like []string{}, or []MyClass{}...
		// So, convert it to a generic array, via reflection.
		ar := []interface{}{}

		valr := reflect.ValueOf(val)
		vlen := valr.Len()
		for i := 0; i < vlen; i++ {
			ar = append(ar, valr.Index(i))
		}

		return ar
	}
	return nil
}

// Merge the fields from object 'src' into object 'dest', recursing for sub-objects
// TODO overwrite isn't enough to fully describe recursive choices
func MergeObjects(dest map[string]interface{}, src map[string]interface{}, overwrite bool) {
	for skey, sval := range src {
		dval, hasKey := dest[skey]
		if !hasKey {
			dest[skey] = sval
		} else {
			dMap, dIsMap := dval.(map[string]interface{})
			sMap, sIsMap := sval.(map[string]interface{})
			if dIsMap && sIsMap {
				// recurse
				MergeObjects(dMap, sMap, overwrite)
			} else if overwrite {
				dest[skey] = sval
			}
		}
	}
}

// Deep (recursive) copy of a JSON object, including sub-maps and sub-arrays.
func DeepCopy(js interface{}) interface{} {
	m, isMap := js.(map[string]interface{})
	a, isArr := js.([]interface{})
	if isMap {
		mcp := map[string]interface{}{}
		for k, v := range m {
			mcp[k] = DeepCopy(v)
		}
		return mcp
	} else if isArr {
		acp := []interface{}{}
		for _, v := range a {
			acp = append(acp, DeepCopy(v))
		}
		return acp
	}
	return js
}

////////////////////////////////////////////////////////////////////////////////

// Fetch a value from js, converting pathStr values like "foo.bar.blah" to js[foo][bar[blah].
// Returns 'dfault' if the target doesn't exist.
func GetNestedValueByString(js interface{}, pathStr string, dfault interface{}) interface{} {
	jsMap, ok := js.(map[string]interface{})
	if !ok {
		return dfault
	}
	path := strings.Split(pathStr, ".")
	return GetNestedValueOrDefault(jsMap, path, dfault)
}

// Convert an array of strings to integers.
// Returns false as the second value if parsing fails.
func StringArrayToInt(vals []string) ([]int64, bool) {
	result := []int64{}
	for _, val := range vals {
		ival, err := strconv.ParseInt(val, 10, 32)
		if err != nil {
			return result, false
		}
		result = append(result, ival)
	}
	return result, true
}

// Assumes 'js' is an array, and 'idx' is a slice string like "[5]" or "[2:4]".
// Returns the appropriate slice of js, or dfault if any assumptions fail.
func GetArraySlice(js interface{}, idx string, dfault interface{}) interface{} {
	arr, ok := js.([]interface{})
	if !ok {
		return dfault
	}
	// skip '[' and ']'
	indicesStr := strings.Split(idx[1:len(idx)-1], ":")
	indices, ok := StringArrayToInt(indicesStr)
	if !ok {
		return dfault
	}
	ilen := len(indices)
	if ilen == 0 || ilen > 2 {
		return dfault
	} else if ilen == 1 {
		//   handle [idx]
		// TODO indicate that it's a single...
		return arr[indices[0]]
	} else {
		//   handle [start:end], should only be at end of path
		//   convert negative indicies (ala Python) to be relative to end-of-array
		if indices[0] < 0 {
			indices[0] = int64(len(arr)) + indices[0]
		}
		if indices[1] < 0 {
			indices[1] = int64(len(arr)) + indices[1]
		}
		return arr[indices[0]:indices[1]]
	}
}

// Dump a JSON object to stdout with a title banner. For debugging.
func DumpJsonObject(js interface{}, title string) {
	jstr, err := json.MarshalIndent(js, "", "  ")
	if err != nil {
		jstr = []byte("NULL")
	}
	os.Stdout.Write([]byte("\n========================================\n"))
	os.Stdout.Write([]byte("\n== " + title + " =\n"))
	os.Stdout.Write(jstr)
	os.Stdout.Write([]byte("\n\n"))
}
