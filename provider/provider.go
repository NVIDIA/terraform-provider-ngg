// Copyright 2025 NVIDIA Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//go:embed provider_conf.json
var ObjectConfigJsonStr string

var (
	RenderedProviderName string // Will be set at build time
	ProviderShortName    string // Will be set at build time
	EnvVarsNamePrefix    string // Will be set at build time
	TfLogFile            string // Will be set at build time
	DefaultUserName      string // Will be set at build time
)

func StringToJsonArray(data string) ([]interface{}, error) {
	//jsObj := map[string]interface{}{}
	jsObj := []interface{}{}
	jsErr := json.Unmarshal([]byte(data), &jsObj)
	return jsObj, jsErr
}

func Base64ToJsonArray(data string) ([]interface{}, error) {
	jsStr, bError := base64.StdEncoding.DecodeString(data)
	if bError != nil {
		return []interface{}{}, bError
	}
	return StringToJsonArray(string(jsStr))
}

func StringToJson(data string) (map[string]interface{}, error) {
	jsObj := map[string]interface{}{}
	jsErr := json.Unmarshal([]byte(data), &jsObj)
	return jsObj, jsErr
}

func Base64ToJson(data string) (map[string]interface{}, error) {
	// NOTE: there are different encoding styles, i.e.:
	//   b64Str := base64.URLEncoding.EncodeToString([]byte(data))
	//   b64Str := base64.StdEncoding.EncodeToString([]byte(data))
	jsStr, bError := base64.StdEncoding.DecodeString(data)
	if bError != nil {
		return map[string]interface{}{}, bError
	}
	return StringToJson(string(jsStr))
}

func OmitJsonObjectFields(val map[string]interface{}, omitList []interface{}) map[string]interface{} {
	appendActionLog(fmt.Sprintf("Omitting (obj) keys: %+v\n", omitList))
	for _, o := range omitList {
		oStr, isStr := o.(string)
		if isStr {
			delete(val, oStr)
		}
	}
	return val
}

func OmitJsonArrayFields(val *[]interface{}, omitList []interface{}) {
	//appendActionLog(fmt.Sprintf("Omitting (array) keys: %+v\n", omitList))
	for idx, elem := range *val {
		eMap, isMap := elem.(map[string]interface{})
		if isMap {
			(*val)[idx] = OmitJsonObjectFields(eMap, omitList)
		}
	}
}
func JsonFieldsWithValue(val map[string]interface{}, omitList []interface{}) bool {
	for _, omitKeyValue := range omitList {
		keyValueMap, isMap := omitKeyValue.(map[string]interface{})
		if isMap {
			key, ok := keyValueMap["key"].(string)
			if !ok {
				continue
			}
			value, ok := keyValueMap["value"]
			if !ok {
				continue
			}
			if val[key] != value {
				return false
			}
		}
	}
	return true
}
func OmitJsonArrayItems(val *[]interface{}, omitList []interface{}) {
	var idx int = 0
	for _, elem := range *val {
		eMap, isMap := elem.(map[string]interface{})
		if !isMap || !JsonFieldsWithValue(eMap, omitList) {
			(*val)[idx] = elem
			idx++
		}
	}
	*val = (*val)[:idx]
}

func timeSuffixToIntSec(tv string) int {
	sz := len(tv)
	l := sz - 1
	if sz < 1 {
		return 0
	}
	mult := 0
	switch tv[sz-1] {
	case 's':
		mult = 1
	case 'm':
		mult = 60
	case 'h':
		mult = 60 * 60
	case 'd':
		mult = 60 * 60 * 24
	default:
		l = sz
		mult = 1
	}
	val := tv[:l]
	i, err := strconv.Atoi(val)
	if err != nil {
		return -1
	}
	return i * mult
}

func appendActionLogInner(msg string) {
	logFile := TfLogFile
	if logFile == "" {
		logFile = "/tmp/tf_provider.log" // fallback only for logging
	}
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		//panic(err)
		return
	}
	defer f.Close()
	id := fmt.Sprintf("gid:%d - ", curGoroutineID())
	if _, err = f.WriteString(id + msg); err != nil {
		//panic(err)
		return
	}
}

func appendActionLog(msg string) {
	if !DoDebugLog {
		return
	}
	appendActionLogInner(msg)
}

func runOpCommand(command string, checkResult bool) (string, error) {
	//var GlobalOpts = CliOpts{}
	//if !LoadAuthConfig(&GlobalOpts) {
	//	return "", fmt.Errorf("Failed to load auth credentials")
	//}
	result := ""
	err := error(nil)
	for r := 0; r <= RetryLimit; r += 1 {
		appendActionLog(fmt.Sprintf("Running OpLang command (retries %d/%d)   ---   command:(( %s ))\n", r, RetryLimit, command))
		result, err = ExecuteOpCommand(&GlobalOpts, command)
		if err == nil {
			if !checkResult {
				return result, err
			}
			err = CheckUpdateResult(result)
			if err == nil {
				return result, err
			} else {
				appendActionLog(fmt.Sprintf("Failed OpLang update (retries %d/%d)   ---   error:(( %s ))\n", r, RetryLimit, err.Error()))
			}
		} else {
			appendActionLog(fmt.Sprintf("Failed OpLang command (retries %d/%d)   ---   error:(( %s ))\n", r, RetryLimit, err.Error()))
		}
	}
	return result, err
}

func runOpCommandToJson(command string) (map[string]interface{}, error) {
	result, err := runOpCommand(command, false)
	if err != nil {
		errOut := fmt.Errorf("Failed to execute op '%s': %s", command, err.Error())
		return nil, errOut
	}

	js := map[string]interface{}{}
	// Parsing/Unmarshalling JSON encoding/json
	err = json.Unmarshal([]byte(result), &js)
	if err != nil {
		errOut := fmt.Errorf("Failed to parse json from command '%s': %s", command, err.Error())
		return nil, errOut
	}
	return js, nil
}

func getNamedObjectFromClassDef(name string, typ string, classJs map[string]interface{}) map[string]interface{} {
	baseKey := ""
	if typ == "nvault_secret" {
		baseKey = "get_secret.secrets"
	} else {
		baseKey = fmt.Sprintf("get_%s_class.%s_classes", typ, typ)
	}
	baseArray, isArray := GetNestedValueOrDefault(classJs, ToKeyPath(baseKey), []interface{}{}).([]interface{})
	if isArray {
		for _, curJs := range baseArray {
			extrName := GetNestedValueOrDefault(curJs, ToKeyPath("name"), "")
			if name == extrName {
				return curJs.(map[string]interface{})
			}
		}
	}
	return map[string]interface{}{}
}

func CheckUpdateResult(result string) error {
	js := map[string]interface{}{}
	err := json.Unmarshal([]byte(result), &js)
	if err != nil {
		return fmt.Errorf("Failed parse json result from resource update %s", err.Error())
	}

	actions := []string{"define", "delete", "update"}
	types := []string{"resource", "alarm", "action", "bot", "file", "integration", "notebook", "configuration", "time_trigger", "principal", "report_template", "dashboard", "secret"}
	for _, act := range actions {
		for _, typ := range types {
			key := act + "_" + typ
			def := GetNestedValueOrDefault(js, ToKeyPath(key), nil)
			if def != nil {
				errKey := key + ".error.message"
				err := GetNestedValueOrDefault(js, ToKeyPath(errKey), nil)
				if (typ == "notebook" || typ == "runbook" || typ == "configuration") && (err == nil || err == "") {
					// have to special-case for notebooks
					err = ""
					errArray := []string{}
					ve, isArray := GetNestedValueOrDefault(js, ToKeyPath(key+".error.validation_errors"), nil).([]interface{})
					if isArray {
						for i, _ := range ve {
							errn, isStr := GetNestedValueOrDefault(js, ToKeyPath(fmt.Sprintf(key+".error.validation_errors.[%d].message", i)), nil).(string)
							if isStr && errn != "" {
								errArray = append(errArray, errn)
							}
							err = strings.Join(errArray, "\n")
						}
					}
				}
				if err == nil || err == "" {
					// success ...
					return nil
				} else {
					errStr := GetInnerErrorStr(CastToString(err))
					// error ...
					return fmt.Errorf("ERROR: %s.\n", errStr)
				}
			}
		}
	}

	// Added error message location...
	execStmtErrors := GetNestedValueOrDefault(js, ToKeyPath("execute_statement_errors"), nil)
	if execStmtErrors != nil {
		errorStrArray := []string{}
		hdArray, isHdArray := execStmtErrors.([]interface{})
		if isHdArray {
			for _, entry := range hdArray {
				entryMap, isMap := entry.(map[string]interface{})
				if !isMap {
					continue
				}

				if errors, ok := entryMap["errors"]; ok {
					errorsArr, isArr := errors.([]interface{})
					if !isArr {
						continue
					}
					for _, msg := range errorsArr {
						msgStr, isStr := msg.(string)
						if isStr {
							errorStrArray = append(errorStrArray, msgStr)
						}
					}
				}
			}
		}
		if len(errorStrArray) > 0 {
			return fmt.Errorf("ERROR: %s.\n", strings.Join(errorStrArray, "\n"))
		}
	}

	return nil
}

// Takes a regex like: "if (?P<if_expr>.*?) then (?P<then_expr>.*?) fi"
// and parses out the named captures (e.g. 'if_expr', 'then_expr')
// into the returned map, with the name as a key, and the match as the value.
func ExtractRegexToMap(expr string, regex string) map[string]interface{} {
	result := map[string]interface{}{}
	re := regexp.MustCompile(regex)
	vals := re.FindStringSubmatch(expr)
	keys := re.SubexpNames()

	vlen := len(vals)
	// skip index 0, which is the entire expression
	for i := 1; i < len(keys); i++ {
		if i < vlen {
			result[keys[i]] = vals[i]
		}
		// XXX else error, capture group didn't match (but need diags passed in)
	}
	return result
}

func ValidateVariableName(name string) bool {
	// match valid variable string names
	matched, _ := regexp.MatchString(`^[_a-zA-Z][_a-zA-Z0-9]*$`, name)
	return matched
}
func ValidateResourceType(name string) bool {
	switch name {
	case "HOST":
		return true
	case "POD":
		return true
	case "CONTAINER":
		return true
	}
	return false
}
func ForceToBool(val interface{}) bool {
	valBool, _ := CastToBoolMaybe(val)
	return valBool
}
func ConvertBoolInt(val interface{}) int {
	valBool, _ := CastToBoolMaybe(val)
	if valBool {
		return 1
	}
	return 0
}

func init() {
	// Read from environment variables first, otherwise use build-time values from -X flags
	if envName := os.Getenv("RENDERED_PROVIDER_NAME"); envName != "" {
		RenderedProviderName = envName
	}
	if envShort := os.Getenv("PROVIDER_SHORT_NAME"); envShort != "" {
		ProviderShortName = envShort
	}
	if envPrefix := os.Getenv("ENV_VARS_NAME_PREFIX"); envPrefix != "" {
		EnvVarsNamePrefix = envPrefix
	}
	if envLogFile := os.Getenv("TF_LOG_FILE"); envLogFile != "" {
		TfLogFile = envLogFile
	}
	if envUser := os.Getenv("DEFAULT_USER_NAME"); envUser != "" {
		DefaultUserName = envUser
	}

	// Set descriptions to support markdown syntax, this will be used in document generation
	// and the language server.
	schema.DescriptionKind = schema.StringMarkdown

	// Customize the content of descriptions when output. For example you can add defaults on
	// to the exported descriptions if present.
	schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
		desc := s.Description
		if s.Default != nil {
			desc += fmt.Sprintf(" Defaults to `%v`.", s.Default)
		}
		return strings.TrimSpace(desc)
	}
}

type VersionRecord struct {
	Valid   bool
	Build   string
	Version string
	Major   int64
	Minor   int64
	Patch   int64
	Error   *error
}

// returns 0,false if either version is invalid
// otherwise, lessThan: -1, equalTo: 0, greaterThan: +1
func CompareVersionRecords(have VersionRecord, want VersionRecord) (gtlteq int, unknown bool) {
	if !have.Valid || !want.Valid {
		return 0, false
	}
	haveVer := []int64{have.Major, have.Minor, have.Patch}
	wantVer := []int64{want.Major, want.Minor, want.Patch}
	for i, want := range wantVer {
		if haveVer[i] < want {
			return -1, true
		}
		if haveVer[i] > want {
			return 1, true
		}
	}
	return 0, true
}

func ExtractVersionData(verStr string) (major int64, minor int64, patch int64, err *error) {
	verRe := regexp.MustCompile(`(\d+)\.(\d+)\.(\d+)`)
	if verRe.MatchString(verStr) {
		match := verRe.FindStringSubmatch(verStr)
		return CastToInt(match[1]), CastToInt(match[2]), CastToInt(match[3]), nil
	}
	major, minor, patch = 0, 0, 0
	erf := fmt.Errorf("Couldn't find version number in string '%s'", verStr)
	err = &erf
	return
}

func GetBackendVersionInfo() (build string, version string, major int64, minor int64, patch int64, err *error) {
	err = nil
	build = "unknown"
	version = "unknown"
	major, minor, patch = 0, 0, 0
	// op> backend_version
	// ... "get_backend_version": "{ \"tag\": \"release-1.2.3-stuff\", \"build_date\": \"Wed_May_18_00:07:11_UTC_2022\" }", ...
	js, opErr := runOpCommandToJson("backend_version")
	if opErr != nil {
		return
	}
	build = GetNestedValueOrDefault(js, ToKeyPath("get_backend_version"), "unknown").(string)
	buildJs := CastToObject(build)
	if buildJs == nil {
		// TODO set error
		return
	}
	version = GetNestedValueOrDefault(buildJs, ToKeyPath("tag"), "unknown").(string)
	if strings.HasPrefix(version, "stable") || strings.HasPrefix(version, "release") || strings.HasPrefix(version, "arm") {
		// parse out '\d+\.\d+.\d+' suffix
		major, minor, patch, err = ExtractVersionData(version)
	} else {
		// dev build, special case
		major, minor, patch = 9999, 9999, 9999
	}
	return
}

func GetBackendVersionInfoStruct() VersionRecord {
	var ver VersionRecord
	ver.Build, ver.Version, ver.Major, ver.Minor, ver.Patch, ver.Error = GetBackendVersionInfo()
	ver.Valid = (ver.Error == nil)
	return ver
}

func ParseVersionString(verStr string) VersionRecord {
	var ver VersionRecord
	ver.Version = verStr
	ver.Major, ver.Minor, ver.Patch, ver.Error = ExtractVersionData(verStr)
	ver.Valid = (ver.Error == nil)
	return ver
}

func dataSourceVersionRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	//client := &http.Client{Timeout: 10 * time.Second}

	// Warning or errors can be collected in a slice type
	var diags diag.Diagnostics
	build, version, major, minor, patch, err := GetBackendVersionInfo()
	if err != nil {
		diags = diag.Errorf("Failed to read backend_version: %s", (*err).Error())
		return diags
	}

	d.Set("build_info", CastToString(build))
	d.Set("version", CastToString(version))
	d.Set("major", major)
	d.Set("minor", minor)
	d.Set("patch", patch)
	// always run
	d.SetId(strconv.FormatInt(time.Now().Unix(), 10))

	return diags
}

func New(version string) func() *schema.Provider {
	return func() *schema.Provider {
		p := &schema.Provider{
			//DataSourcesMap: map[string]*schema.Resource{
			//	ProviderShortName + "_datasource": dataSourceObject(),
			//},
			ResourcesMap: map[string]*schema.Resource{
				ProviderShortName + "_action":          ResourceObject(ObjectConfigJsonStr, "action"),
				ProviderShortName + "_alarm":           ResourceObject(ObjectConfigJsonStr, "alarm"),
				ProviderShortName + "_time_trigger":    ResourceObject(ObjectConfigJsonStr, "time_trigger"),
				ProviderShortName + "_bot":             ResourceObject(ObjectConfigJsonStr, "bot"),
				ProviderShortName + "_file":            ResourceObject(ObjectConfigJsonStr, "file"),
				ProviderShortName + "_integration":     ResourceObject(ObjectConfigJsonStr, "integration"),
				ProviderShortName + "_notebook":        ResourceObject(ObjectConfigJsonStr, "notebook"),
				ProviderShortName + "_runbook":         ResourceObject(ObjectConfigJsonStr, "notebook"), // alias <name>_runbook to <name>_notebook
				ProviderShortName + "_principal":       ResourceObject(ObjectConfigJsonStr, "principal"),
				ProviderShortName + "_resource":        ResourceObject(ObjectConfigJsonStr, "resource"),
				ProviderShortName + "_system_settings": ResourceObject(ObjectConfigJsonStr, "system_settings"),
				ProviderShortName + "_report_template": ResourceObject(ObjectConfigJsonStr, "report_template"),
				ProviderShortName + "_dashboard":       ResourceObject(ObjectConfigJsonStr, "dashboard"),
				ProviderShortName + "_nvault_secret":   ResourceObject(ObjectConfigJsonStr, "nvault_secret"),
			},
			DataSourcesMap: map[string]*schema.Resource{
				ProviderShortName + "_version": &schema.Resource{
					ReadContext: dataSourceVersionRead,
					Schema: map[string]*schema.Schema{
						"build_info": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"version": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"major": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"minor": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"patch": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			Schema: map[string]*schema.Schema{
				"url": {
					Type:     schema.TypeString,
					Required: true,
					ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
						if !ValidateApiUrl(val.(string)) {
							errs = append(errs, fmt.Errorf("%q must be a valid URL,\n but got: %s", key, val.(string)))
						}
						return
					},
					DefaultFunc: schema.EnvDefaultFunc(EnvVarsNamePrefix+"_URL", nil),
					Description: "Customer-specific URL for the " + RenderedProviderName + " API server.",
				},
				"token": {
					Type:        schema.TypeString,
					Optional:    true,
					Sensitive:   true,
					DefaultFunc: schema.EnvDefaultFunc(EnvVarsNamePrefix+"_TOKEN", nil),
					Description: "Customer/user-specific authorization token for the " + RenderedProviderName + " API server. May be provided via `" + EnvVarsNamePrefix + "_TOKEN` env variable.",
				},
				"retries": {
					Type:        schema.TypeInt,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc(EnvVarsNamePrefix+"_RETRIES", nil),
					Description: "Number of retries for API calls, in case of e.g. transient network failures.",
				},
				"debug": {
					Type:        schema.TypeBool,
					Optional:    true,
					DefaultFunc: schema.EnvDefaultFunc(EnvVarsNamePrefix+"_DEBUG", nil),
					Description: "Debug logging to `" + TfLogFile + "`.",
				},
				"min_version": {
					Type:        schema.TypeString,
					Optional:    true,
					Description: "Minimum version required on the " + RenderedProviderName + " backend (API server).",
				},
			},
		}

		p.ConfigureContextFunc = configure(version, p)

		return p
	}
}

type apiClient struct {
	// Add whatever fields, client or connection info, etc. here
	// you would need to setup to communicate with the upstream
	// API.
}

func configure(version string, p *schema.Provider) func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	return func(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
		AuthUrl = d.Get("url").(string)
		token, hasToken := d.GetOk("token")

		var diags diag.Diagnostics = nil

		if hasToken {
			SetAuth(&GlobalOpts, AuthUrl, token.(string))
		} else {
			GlobalOpts.Url = AuthUrl
			if !LoadAuthConfig(&GlobalOpts) {
				return nil, diag.Errorf("Failed to load auth credentials file.\n" + GetManualAuthMessage(&GlobalOpts))
			}
			if !selectAuth(&GlobalOpts, AuthUrl) {
				return nil, diag.Errorf("Failed to load auth credentials for %s\n"+GetManualAuthMessage(&GlobalOpts), AuthUrl)
			}
		}

		retries, hasRetry := d.GetOk("retries")
		if hasRetry {
			RetryLimit = retries.(int)
		} else {
			RetryLimit = 0
		}

		debugLog, hasDebugLog := d.GetOk("debug")
		if hasDebugLog {
			DoDebugLog = debugLog.(bool)
		}

		minVer, hasMinVer := d.GetOk("min_version")
		if hasMinVer {
			var diags diag.Diagnostics
			_, version, major, minor, patch, err := GetBackendVersionInfo()
			if err != nil {
				diags = diag.Errorf("Failed to read backend_version: %s", (*err).Error())
				return nil, diags
			}
			minMajor, minMinor, minPatch, err := ExtractVersionData(minVer.(string))
			if err != nil {
				diags = diag.Errorf("Failed to parse min_version: %s", (*err).Error())
				return nil, diags
			}
			wantVer := []int64{minMajor, minMinor, minPatch}
			haveVer := []int64{major, minor, patch}
			verOk := true
			for i, want := range wantVer {
				if haveVer[i] < want {
					verOk = false
					break
				}
				if haveVer[i] > want {
					break
				}
			}
			if !verOk {
				diags = diag.Errorf("Backend version '%s' (%d, %d, %d) does not meet min_version: '%s' (%d, %d, %d)", version, major, minor, patch, minVer.(string), minMajor, minMinor, minPatch)
				return nil, diags
			}
		}

		return &apiClient{}, diags
	}
}

////////////////////////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////////////////////////

func ResourceObject(configJsStr string, key string) *schema.Resource {
	params := map[string]*schema.Schema{}

	objects := map[string]interface{}{}
	// Parsing/Unmarshalling JSON encoding/json
	err := json.Unmarshal([]byte(configJsStr), &objects)
	if err != nil {
		WriteMsg("WARNING: Failed to parse JSON config from ResourceObject().\n")
		return nil
	}
	object := GetNestedValueOrDefault(objects, ToKeyPath(key), nil)
	if object == nil {
		WriteMsg("WARNING: Failed to parse JSON config from ResourceObject(%s).\n", key)
		return nil
	}
	attributes := GetNestedValueOrDefault(object, ToKeyPath("attributes"), map[string]interface{}{}).(map[string]interface{})
	for k, _ := range attributes {
		if strings.HasPrefix(k, "#") {
			delete(attributes, k)
		}
	}
	primary := "name"
	for k, attrs := range attributes {
		// internal objects, i.e. components of compound fields
		internal := GetNestedValueOrDefault(attrs, ToKeyPath("internal"), false).(bool)
		if internal {
			continue
		}

		sch := &schema.Schema{}
		maybeAddValidateFunc(sch, key, k)

		description := CastToString(GetNestedValueOrDefault(objects, ToKeyPath("docs.attributes."+k), ""))
		sch.Description = description

		attrMap := attrs.(map[string]interface{})
		typ := GetNestedValueOrDefault(attrMap, ToKeyPath("type"), "string")
		switch typ {
		case "command":
			sch.Type = schema.TypeString
			sch.DiffSuppressFunc = func(k, old, nu string, d *schema.ResourceData) bool {
				// ignore whitespace changes in command strings
				if strings.ReplaceAll(old, " ", "") == strings.ReplaceAll(nu, " ", "") {
					return true
				}
				return false
			}
		case "time_s":
			sch.Type = schema.TypeString
			// special case for circuit_breaker (to be tested after refactoring this area)
			sch.DiffSuppressFunc = func(k, old, nu string, d *schema.ResourceData) bool {
				oldT := timeSuffixToIntSec(old)
				nuT := timeSuffixToIntSec(nu)
				appendActionLog(fmt.Sprintf("time_s DiffSuppressFunc: diffing (%s)=(%d) and (%s)=(%d)", old, oldT, nu, nuT))
				if oldT == nuT {
					return true
				}
				return false
			}
		case "b64json":
			sch.Type = schema.TypeString
			// special case for notebook JSON data, or cells
			sch.DiffSuppressFunc = func(k, old, nu string, d *schema.ResourceData) bool {
				//appendActionLog(fmt.Sprintf("notebook.params DiffSuppressFunc(%v : %v) \n", key, k))
				//appendActionLog(fmt.Sprintf("NormalizeNotebookJson() PreParse Nu:  '%s'\n", nu))
				//appendActionLog(fmt.Sprintf("NormalizeNotebookJson() PreParse Old: '%s'\n", old))
				if old == "" && nu == "" {
					return true
				}
				// NOTE: notebook.params has "match_null", so it's handled below
				if k == "cells" {
					oldArr, oldErr := StringToJsonArray(old)
					nuArr, nuErr := StringToJsonArray(nu)
					if oldErr != nil || nuErr != nil {
						return false
					}
					NormalizeNotebookCells(&nuArr)
					NormalizeNotebookCells(&oldArr)
					if reflect.DeepEqual(oldArr, nuArr) {
						return true
					}
				}
				if k == "data" {
					oldJs, oldErr := StringToJson(old)
					nuJs, nuErr := StringToJson(nu)
					if oldErr != nil || nuErr != nil {
						return false
					}
					// special case top-level notebook "enabled" which may be returned by old backends
					delete(nuJs, "enabled")
					delete(oldJs, "enabled")
					NormalizeNotebookJson(nuJs, attributes)
					NormalizeNotebookJson(oldJs, attributes)
					//appendActionLog(fmt.Sprintf("NormalizeNotebookJson() NuJs:  '%s'\n", CastToString(nuJs)))
					//appendActionLog(fmt.Sprintf("NormalizeNotebookJson() OldJs: '%s'\n", CastToString(oldJs)))
					//appendActionLog(fmt.Sprintf("notebook.data DiffSuppressFunc, new: %+v \n", nuJs))
					//appendActionLog(fmt.Sprintf("notebook.data DiffSuppressFunc, old: %+v \n", oldJs))
					if reflect.DeepEqual(oldJs, nuJs) {
						return true
					}
				}
				if key == "report_template" {
					switch k {
					case "blocks":
						oldJsonEncoded, err := base64.StdEncoding.DecodeString(old)
						if err != nil {
							return false
						}
						return string(oldJsonEncoded) == nu

					case "links":
						var conf string
						err := json.Unmarshal([]byte(old), &conf)
						if err != nil {
							return false
						}
						return string(conf) == nu
					}
				}
				if key == "dashboard" {
					if k == "groups" || k == "values" {
						var conf string
						err := json.Unmarshal([]byte(old), &conf)
						if err != nil {
							return false
						}
						return string(conf) == nu
					}
				}
				return false
			}
			// TODO warn if "data.force_set[i]" fields are present
			//sch.ValidateFunc = func(val interface{}, key string) (warns []string, errs []error) {
			//	v := val.(int)
			//	if v <= 0 {
			//		errs = append(errs, fmt.Errorf("%q must be > 0, got: %d", key, v))
			//	}
			//	return
			//}
		case "string":
			sch.Type = schema.TypeString
		case "string[]":
			sch.Type = schema.TypeList
			sch.Elem = &schema.Schema{
				Type: schema.TypeString,
			}
		case "string_set":
			sch.Type = schema.TypeList
			sch.Elem = &schema.Schema{
				Type: schema.TypeString,
			}
			sch.DiffSuppressFunc = func(k, old, nu string, d *schema.ResourceData) bool {
				//appendActionLog(fmt.Sprintf("string_set DiffSuppressFunc,     k: '%+v'     new: '%+v'     old: '%+v'     d: '%+v' \n", k, old, nu, d))
				// special handling because DiffSuppressFunc doesn't natively work for
				// list-type attributes: https://github.com/hashicorp/terraform-plugin-sdk/issues/477#issue-640263603
				lastDotIndex := strings.LastIndex(k, ".")
				kPrefix := k
				if lastDotIndex != -1 {
					kPrefix = string(k[:lastDotIndex])
				}
				oldData, newData := d.GetChange(kPrefix)
				if oldData == nil || newData == nil {
					//appendActionLog(fmt.Sprintf("string_set DiffSuppressFunc (one nil),   oldData: '%+v'   newData: '%+v'\n", oldData, newData))
					return false
				}
				if strings.HasSuffix(k, ".#") {
					appendActionLog(fmt.Sprintf("string_set DiffSuppressFunc (checking size),   oldData: '%+v'   newData: '%+v'\n", old, nu))
					if old != nu {
						return false
					}
				}
				oldSortedList := SortListByStrVal(CastToArray(oldData)) // from []any to []string
				newSortedList := SortListByStrVal(CastToArray(newData))
				//appendActionLog(fmt.Sprintf("string_set DiffSuppressFunc (lists pre-sort),   oldArray: '%+v'   newArray: '%+v'\n", oldData, newData))
				//appendActionLog(fmt.Sprintf("string_set DiffSuppressFunc (sorted lists),   oldList: '%+v'   newList: '%+v'\n", oldSortedList, newSortedList))
				return reflect.DeepEqual(oldSortedList, newSortedList)
			}
		case "bool":
			sch.Type = schema.TypeBool
		case "intbool":
			// special handling to/from backend ("1"/"0")
			sch.Type = schema.TypeBool
		case "float":
			sch.Type = schema.TypeFloat
		case "int":
			sch.Type = schema.TypeInt
		case "unsigned":
			sch.Type = schema.TypeInt
			// non-negative validator
			sch.ValidateFunc = func(val interface{}, key string) (warns []string, errs []error) {
				v := val.(int)
				if v <= 0 {
					errs = append(errs, fmt.Errorf("%q must be > 0, got: %d", key, v))
				}
				return
			}
		case "label":
			sch.Type = schema.TypeString
			// ValidateVariableName()
			sch.ValidateFunc = func(val interface{}, key string) (warns []string, errs []error) {
				re := regexp.MustCompile("^[a-zA-Z0-9_]*$")
				v, isStr := val.(string)
				if !isStr || (!re.MatchString(v)) {
					errs = append(errs, fmt.Errorf("%q must be an alphanumeric/underscore string, got: '%+v'", key, val))
				} else {
					res := regexp.MustCompile("^[a-zA-Z_]")
					if !res.MatchString(v) {
						errs = append(errs, fmt.Errorf("%q must start with a letter or underscore, got: '%+v'", key, val))
					}
				}
				return
			}
		case "resource":
			sch.Type = schema.TypeString
			// TODO ValidateResourceType() "^(HOST|POD|CONTAINER)$"
		}
		sch.Optional = GetNestedValueOrDefault(attrMap, ToKeyPath("optional"), false).(bool)
		sch.Required = GetNestedValueOrDefault(attrMap, ToKeyPath("required"), false).(bool)
		sch.Computed = GetNestedValueOrDefault(attrMap, ToKeyPath("computed"), false).(bool)
		sch.ForceNew = GetNestedValueOrDefault(attrMap, ToKeyPath("forcenew"), false).(bool)
		deprecated := GetNestedValueOrDefault(attrMap, ToKeyPath("deprecated"), false).(bool)
		deprField := GetNestedValueOrDefault(attrMap, ToKeyPath("deprecated_for"), "").(string)
		replField := GetNestedValueOrDefault(attrMap, ToKeyPath("replaces"), "").(string)
		conflicts := GetNestedValueOrDefault(attrMap, ToKeyPath("conflicts"), []interface{}{}).([]interface{})
		if deprecated {
			sch.Deprecated = fmt.Sprintf("Field '%s' is obsolete.", k)
			sch.Description = "**Deprecated** " + sch.Deprecated + " " + description
		}
		if deprField != "" {
			sch.Deprecated = fmt.Sprintf("Please use '%s' instead.", deprField)
			sch.ConflictsWith = []string{deprField}
			sch.Description = "**Deprecated** " + sch.Deprecated + " " + description
			// XXX does this need diff suppression?
			//sch.DiffSuppressFunc = func(k, old, nu string, d *schema.ResourceData) bool { }
		}
		if len(conflicts) > 0 {
			confArr := []string{}
			for _, c := range conflicts {
				confArr = append(confArr, c.(string))
			}
			sch.ConflictsWith = confArr
		}
		replacesField := GetNestedValueOrDefault(attrMap, ToKeyPath("replaces"), "").(string)
		if replacesField != "" {
			sch.ConflictsWith = []string{replacesField}
		}
		//WriteMsg("WARNING: JSON config from ResourceObject(%s) %s.Optional = %+v.\n", key, k, sch.Optional)
		//WriteMsg("WARNING: JSON config from ResourceObject(%s) %s.Required = %+v.\n", key, k, sch.Required)
		//WriteMsg("WARNING: JSON config from ResourceObject(%s) %s.Computed = %+v.\n", key, k, sch.Computed)
		//defowlt := GetNestedValueOrDefault(attrMap, ToKeyPath("value"), nil)
		sch, defowlt := SetAttributeDefaultValue(attrMap, sch)
		//appendActionLogInner(fmt.Sprintf("NOTE: DEFAULT ResourceObject(%s) %s.Default = %+v.\n", key, k, defowlt))

		suppressNullDiffRegex, isStr := GetNestedValueOrDefault(attrMap, ToKeyPath("suppress_null_regex"), nil).(string)
		if isStr {
			sch.DiffSuppressFunc = func(k, old, nu string, d *schema.ResourceData) bool {
				appendActionLog(fmt.Sprintf("suppressNullDiff check: '%s': '%s' -- vs -- '%s'\n", suppressNullDiffRegex, old, nu))
				if old == nu {
					return true
				}
				if nu == "" {
					matched, _ := regexp.MatchString(suppressNullDiffRegex, old)
					if matched {
						return true
					}
				}
				return false
			}
		}
		matchNull := GetNestedValueOrDefault(attrMap, ToKeyPath("match_null"), nil)
		if matchNull != nil {
			sch.DiffSuppressFunc = func(k, old, nu string, d *schema.ResourceData) bool {
				//appendActionLog(fmt.Sprintf("suppressNullMatch check: '%v': '%s' -- vs -- '%s'\n", matchNull, old, nu))
				if old == nu {
					return true
				}
				if old == "" && nu == matchNull {
					return true
				}
				if nu == "" && old == matchNull {
					return true
				}

				if k == "params" || k == "external_params" {
					oldJs, oldErr := StringToJsonArray(old)
					nuJs, nuErr := StringToJsonArray(nu)
					//appendActionLog(fmt.Sprintf("notebook.params DiffSuppressFunc(), PRE  old: %+v \n", oldJs))
					//appendActionLog(fmt.Sprintf("notebook.params DiffSuppressFunc(), PRE  nu: %+v \n", nuJs))
					if oldErr != nil || nuErr != nil {
						return false
					}
					if k == "params" {
						AddNotebookParamsFields(nuJs)
						AddNotebookParamsFields(oldJs)
					} else {
						AddNotebookExternalParamsFields(nuJs)
						AddNotebookExternalParamsFields(oldJs)
					}
					//appendActionLog(fmt.Sprintf("notebook.params DiffSuppressFunc(), POST old: %+v \n", oldJs))
					//appendActionLog(fmt.Sprintf("notebook.params DiffSuppressFunc(), POST nu: %+v \n", nuJs))
					if reflect.DeepEqual(oldJs, nuJs) {
						return true
					}
				}

				return false
			}
		}

		associatedSettings := map[string]string{
			"notebook_ad_hoc_approval_request_enabled":      "runbook_ad_hoc_approval_request_enabled",
			"notebook_approval_request_expiry_time":         "runbook_approval_request_expiry_time",
			"notebook_run_approval_expiry_time":             "run_approval_expiry_time",
			"parallel_notebook_runs_fired_by_time_triggers": "parallel_runs_fired_by_time_triggers",
			"runbook_ad_hoc_approval_request_enabled":       "notebook_ad_hoc_approval_request_enabled",
			"runbook_approval_request_expiry_time":          "notebook_approval_request_expiry_time",
			"run_approval_expiry_time":                      "notebook_run_approval_expiry_time",
			"parallel_runs_fired_by_time_triggers":          "parallel_notebook_runs_fired_by_time_triggers",
		}
		if deprField != "" {
			sch.DiffSuppressFunc = func(k, old, nu string, d *schema.ResourceData) bool {
				_, preferredSettingNewValue := d.GetChange(associatedSettings[k])

				if old == nu {
					return true
				}

				switch typ {
				case "int":
					if nu == fmt.Sprintf("%v", defowlt.(float64)) && (preferredSettingNewValue.(int) != int(defowlt.(float64))) {
						return true
					}
				case "bool":
					if nu == fmt.Sprintf("%v", defowlt.(bool)) && (preferredSettingNewValue.(bool) != defowlt.(bool)) {
						return true
					}
				default:
					if defowlt != nil && nu == defowlt.(string) && (preferredSettingNewValue.(string) != defowlt.(string)) {
						return true
					}
				}

				return false
			}
		}

		if replField != "" {
			sch.DiffSuppressFunc = func(k, old, nu string, d *schema.ResourceData) bool {
				if old == nu {
					return true
				}

				_, deprecatedSettingNewValue := d.GetChange(associatedSettings[k])
				switch typ {
				case "int":
					if nu == fmt.Sprintf("%v", defowlt.(float64)) && (deprecatedSettingNewValue.(int) != int(defowlt.(float64))) {
						return true
					}
				case "bool":
					if nu == fmt.Sprintf("%v", defowlt.(bool)) && (deprecatedSettingNewValue.(bool) != defowlt.(bool)) {
						return true
					}
				default:
					if defowlt != nil && nu == defowlt.(string) && deprecatedSettingNewValue.(string) != defowlt.(string) {
						return true
					}
				}

				return false
			}
		}
		// NOTE: This actually messes up the file objects. Need a suppress function that's just for acceptance test comparisions.
		//notStored, isBool := GetNestedValueOrDefault(attrMap, ToKeyPath("not_stored"), nil).(bool)
		//if isBool && notStored {
		//	sch.DiffSuppressFunc = func(k, old, nu string, d *schema.ResourceData) bool {
		//		//appendActionLog(fmt.Sprintf("Not Stored Value: '%s': '%s' -- vs -- '%s'\n", k, old, nu))
		//		if old == nu || nu == "" || old == "" {
		//			return true
		//		}
		//		return false
		//	}
		//}

		if GetNestedValueOrDefault(attrMap, ToKeyPath("primary"), false).(bool) {
			primary = k
		}
		params[k] = sch

	}

	objDescription := CastToString(GetNestedValueOrDefault(objects, ToKeyPath("docs.objects."+key), ""))
	objectDef, _ := object.(map[string]interface{})

	return &schema.Resource{
		Description: RenderedProviderName + " " + key + ". " + objDescription,

		CreateContext: ResourceObjectCreate(key, primary, attributes, objectDef),
		ReadContext:   ResourceObjectRead(key, attributes, objectDef),
		UpdateContext: ResourceObjectUpdate(key, attributes, objectDef),
		DeleteContext: ResourceObjectDelete(key, objectDef),
		Importer:      &schema.ResourceImporter{State: schema.ImportStatePassthrough},

		Schema: params,
	}

}

func AddNotebookParamsFields(params []interface{}) {
	for _, v := range params {
		theMap, isMap := v.(map[string]interface{})
		// XXX complain if it's not a map...
		if isMap {
			_, hasRequired := theMap["required"]
			if !hasRequired {
				theMap["required"] = true
			}

			_, hasExport := theMap["export"]
			if !hasExport {
				theMap["export"] = false
			}
			val, hasValue := theMap["value"]
			if !hasValue {
				theMap["value"] = ""
			} else {
				_, isStr := val.(string)
				if !isStr {
					theMap["value"] = CastToString(val)
				}
			}
			description, hasDescription := theMap["description"]
			if !hasDescription {
				theMap["description"] = ""
			} else {
				_, isStr := description.(string)
				if !isStr {
					theMap["description"] = CastToString(description)
				}
			}
		}
	}
}

func AddNotebookExternalParamsFields(externalParams []interface{}) {
	for _, v := range externalParams {
		theMap, isMap := v.(map[string]interface{})
		if isMap {
			_, hasValue := theMap["value"]
			if !hasValue {
				theMap["value"] = ""
			}

			_, hasExport := theMap["export"]
			if !hasExport {
				theMap["export"] = false
			}

			description, hasDescription := theMap["description"]
			if !hasDescription {
				theMap["description"] = ""
			} else {
				_, isStr := description.(string)
				if !isStr {
					theMap["description"] = CastToString(description)
				}
			}
		}
	}
}

func NormalizeNotebookJsonArray(arr []interface{}) {
	for _, v := range arr {
		theMap, isMap := v.(map[string]interface{})
		if isMap {
			NormalizeNotebookJson(theMap, nil)
		}
	}
}

func NormalizeNotebookJson(object map[string]interface{}, attributes map[string]interface{}) {
	toRemove := map[string]bool{}
	if attributes != nil {
		skip_diff, ok := GetNestedValueOrDefault(attributes, ToKeyPath("data.skip_diff"), []interface{}{}).([]interface{})
		if ok {
			for _, k := range skip_diff {
				toRemove[CastToString(k)] = true
			}
		}
	}
	for k, v := range object {
		arr, isArray := v.([]interface{})
		if toRemove[CastToString(k)] {
			appendActionLog(fmt.Sprintf("NormalizeNotebookJson() toRemove: '%+v'\n", k))
			delete(object, k)
		} else if isArray {
			if k == "params" {
				AddNotebookParamsFields(arr)
			}

			if k == "external_params" {
				AddNotebookExternalParamsFields(arr)
			}

			// remove empty lists (e.g. external_params)
			if len(arr) == 0 {
				appendActionLog(fmt.Sprintf("NormalizeNotebookJson() removing empty array: '%+v'\n", k))
				delete(object, k)
			} else {
				// NOTE: In future, may need to sort nested non-ordinal lists (ala top-level allowed_entities).
				NormalizeNotebookJsonArray(arr)
			}
		} else {
			theMap, isMap := v.(map[string]interface{})
			if isMap {
				NormalizeNotebookJson(theMap, nil)
			} else {
				if k == "external_params" && v == nil {
					delete(object, k)
				}
			}
		}
	}

	if attributes != nil {
		_, hasExtParams := object["external_params"]
		if !hasExtParams {
			//appendActionLog(fmt.Sprintf("NormalizeNotebookJson() Adding: 'external_params'\n"))
			object["external_params"] = []interface{}{}
		}

		// NOTE: Currently interactive_state only contains transient data
		delete(object, "interactive_state")
		//_, hasInterState := object["interactive_state"]
		//if !hasInterState {
		//	object["interactive_state"] = map[string]interface{}{}
		//}

		_, hasFav := object["is_favorite"]
		if !hasFav {
			//appendActionLog(fmt.Sprintf("NormalizeNotebookJson() Adding: 'external_params'\n"))
			object["is_favorite"] = false
		}

	}

}

func NormalizeNotebookCells(cells *[]interface{}) {
	omitList := []interface{}{"id", "dynamic_cell_fields", "dynamic_fields"}
	OmitJsonArrayFields(cells, omitList)

	for _, v := range *cells {
		vmap := v.(map[string]interface{})
		ctyp := GetNestedValueOrDefault(vmap, ToKeyPath("type"), "").(string)
		content := GetNestedValueOrDefault(vmap, ToKeyPath("content"), nil)
		if ctyp == "OP_LANG" {
			vmap["op"] = content
			delete(vmap, "type")
			delete(vmap, "content")
		} else if ctyp == "MARKDOWN" {
			vmap["md"] = content
			delete(vmap, "type")
			delete(vmap, "content")
		}
		enabled := GetNestedValueOrDefault(vmap, ToKeyPath("enabled"), nil)
		name := GetNestedValueOrDefault(vmap, ToKeyPath("name"), nil)

		if enabled == nil {
			vmap["enabled"] = true
		}
		if name == nil {
			vmap["name"] = "unnamed"
		}

		backendVersion := GetBackendVersionInfoStruct()
		if IsSecretAwareSupported(backendVersion) {
			secret_aware := GetNestedValueOrDefault(vmap, ToKeyPath("secret_aware"), nil)
			// set secret_aware only if backend version >= 28.1 && backend_version != 28.3
			if secret_aware == nil {
				vmap["secret_aware"] = false
			}
		} else {
			// Explicitly remove secret_aware if backend version doesn't support it
			delete(vmap, "secret_aware")
		}
	}
}

func IsSecretAwareSupported(backendVersion VersionRecord) bool {
	if !backendVersion.Valid {
		var build struct {
			Tag string `json:"tag"`
		}
		if err := json.Unmarshal([]byte(backendVersion.Build), &build); err == nil {
			return strings.HasPrefix(build.Tag, "private-") ||
				strings.HasPrefix(build.Tag, "arm-private-") ||
				strings.HasPrefix(build.Tag, "master-")
		}
	}

	if backendVersion.Major > 28 {
		return true
	}
	if backendVersion.Major == 28 {
		return (backendVersion.Minor == 1 && backendVersion.Patch >= 54) ||
			(backendVersion.Minor == 2 && backendVersion.Patch >= 4) ||
			backendVersion.Minor > 3
	}
	return false // Covers Major < 28
}

func EscapeString(val interface{}) string {
	str := fmt.Sprintf("%s", val)

	quoted_str := strconv.Quote(str)

	return quoted_str[1 : len(quoted_str)-1]
}

func SortListByStrVal(val []interface{}) []interface{} {
	sortedCopy := make([]interface{}, len(val), len(val))
	copy(sortedCopy, val)
	sort.Slice(sortedCopy, func(i, j int) bool {
		s1, s2 := "\"\"", "\"\""
		if sortedCopy[i] != nil {
			s1 = fmt.Sprintf("\"%s\"", EscapeString(sortedCopy[i]))
		}
		if sortedCopy[j] != nil {
			s2 = fmt.Sprintf("\"%s\"", EscapeString(sortedCopy[j]))
		}
		return s1 < s2
	})
	return sortedCopy
}

// AttrValueDefault returns the default value for a given attribute type
// Used for both schema defaults and when no value is provided in configuration
func AttrValueDefault(attrTyp string) interface{} {
	switch attrTyp {
	case "string", "command", "time_s", "label", "resource", "b64json":
		return ""
	case "bool", "intbool":
		return false
	case "int":
		return int(0)
	case "unsigned":
		return uint(0)
	case "float":
		return float64(0)
	case "string[]", "string_set":
		return []string{}
	default:
		return ""
	}
}

// SetAttributeDefaultValue sets the default value for a schema attribute based on the attribute map configuration.
// If the attribute is required or computed, no default is set. Also, defaults are not set for list or set types
// as they're not supported by Terraform. Otherwise, it will use either the explicitly specified default
// from the attribute map, or fall back to a type-specific default value if none is specified.
func SetAttributeDefaultValue(attrMap map[string]interface{}, sch *schema.Schema) (*schema.Schema, interface{}) {
	required := GetNestedValueOrDefault(attrMap, ToKeyPath("required"), false).(bool)
	computed := GetNestedValueOrDefault(attrMap, ToKeyPath("computed"), false).(bool)
	attrTyp := GetNestedValueOrDefault(attrMap, ToKeyPath("type"), "string").(string)

	// Don't set default if the field is required or computed
	if required || computed {
		return sch, nil
	}

	// Don't set default for list or set types (string[] or string_set)
	if attrTyp == "string[]" || attrTyp == "string_set" {
		return sch, nil
	}

	defowlt := GetNestedValueOrDefault(attrMap, ToKeyPath("default"), nil)

	if defowlt == nil {
		// If no default is specified, use the type-specific default
		defowlt = AttrValueDefault(attrTyp)
	}

	sch.Default = defowlt

	return sch, defowlt
}

func attrValueString(typ string, key string, val interface{}, attrs map[string]interface{}) string {
	strVal := ""
	attrTyp := GetNestedValueOrDefault(attrs, ToKeyPath(key+".type"), "string").(string)
	optional := GetNestedValueOrDefault(attrs, ToKeyPath(key+".optional"), false).(bool)
	switch attrTyp {
	case "command":
		if optional && val == "" {
			strVal = "\"\""
		} else {
			strVal = fmt.Sprintf("%s", val)
		}
	case "time_s":
		strVal = fmt.Sprintf("%s", val)
	case "b64json":
		jsStr, isStr := val.(string)
		if !isStr {
			jsStr = ""
		}
		strVal = fmt.Sprintf("\"%s\"", base64.StdEncoding.EncodeToString([]byte(jsStr)))
	case "string":
		strVal = fmt.Sprintf("\"%s\"", EscapeString(val))
	case "string[]":
		valArr, isArr := val.([]interface{})
		listStr := ""
		sep := ""
		if isArr {
			for _, v := range valArr {
				if v == nil {
					listStr = listStr + fmt.Sprintf("%s\"\"", sep)
				} else {
					listStr = listStr + fmt.Sprintf("%s\"%s\"", sep, EscapeString(v))
				}
				sep = ", "
			}
		}
		return "[ " + listStr + " ]"
	case "string_set":
		valArr, isArr := val.([]interface{})
		listStr := ""
		sep := ""
		if isArr {
			for _, v := range valArr {
				if v == nil {
					listStr = listStr + fmt.Sprintf("%s\"\"", sep)
				} else {
					listStr = listStr + fmt.Sprintf("%s\"%s\"", sep, EscapeString(v))
				}
				sep = ", "
			}
		}
		return "[ " + listStr + " ]"
	case "bool":
		if ForceToBool(val) {
			strVal = fmt.Sprintf("true")
		} else {
			strVal = fmt.Sprintf("false")
		}
	case "intbool": // special handling to/from backend ("1"/"0")
		strVal = fmt.Sprintf("%d", ConvertBoolInt(val))
	case "float":
		strVal = fmt.Sprintf("%f", val)
	case "int":
		strVal = fmt.Sprintf("%d", val)
	case "unsigned":
		strVal = fmt.Sprintf("%d", val)
	case "label":
		strVal = fmt.Sprintf("\"%s\"", val)
	case "resource":
		strVal = fmt.Sprintf("\"%s\"", val)
	}
	return strVal
}

func setFieldViaOp(typ string, attrs map[string]interface{}, name string, key string, val interface{}) diag.Diagnostics {
	var diags diag.Diagnostics

	valStr := attrValueString(typ, key, val, attrs)
	op := fmt.Sprintf("%s.%s = %s", name, key, valStr)

	if typ == "dashboard" {
		isPrimary := GetNestedValueOrDefault(attrs, ToKeyPath(key+".primary"), false).(bool)
		if isPrimary {
			appendActionLog(fmt.Sprintf("Skipping setting %s field %s...\n", typ, key))
			return nil
		} else {
			if key == "groups" || key == "values" {
				op = fmt.Sprintf("%s.%s = %s", name, key, val)
			}
		}
	}

	appendActionLog(fmt.Sprintf("Setting %s field: '%s'.'%s' :: %+v\n", typ, name, key, val))

	// TODO Let alias to be a list of fallbacks for versioning,
	//   or have alternate ObjectConfigJsonStr based on backend version,
	//   or let backend return ObjectConfigJsonStr to use.
	alias, isStr := GetNestedValueOrDefault(attrs, ToKeyPath(key+".alias_out"), nil).(string)
	if isStr {
		//appendActionLog(fmt.Sprintf("Setting %s aliased field: '%s'->'%s'.'%s' :: %+v\n", typ, name, alias, key, val))
		op = fmt.Sprintf("%s.%s = %s", name, alias, valStr)
	}

	appendActionLog(fmt.Sprintf("Setting with op statement... '%s'\n", op))
	result, err := runOpCommand(op, true)
	if err != nil {
		diags = diag.Errorf("Failed to set %s %s.%s: %s", typ, name, key, err.Error())
		appendActionLog(fmt.Sprintf("Failed to set %s %s.%s: %s\nval: (( %+v ))\nop-statement: %s\n", typ, name, key, val, err.Error(), op))
		return diags
	}
	err = CheckUpdateResult(result)
	if err != nil {
		diags = diag.Errorf("Failed to update %s %s.%s: %s", typ, name, key, err.Error())
		return diags
	}
	return nil
}

func getRemoteFileAttr(name string, key string) string {
	pathAttrCmd := fmt.Sprintf("%s.%s", name, key)
	pathJson, err := runOpCommandToJson(pathAttrCmd)
	if err != nil {
		return ""
	}
	uri, isStr := GetNestedValueOrDefault(pathJson, ToKeyPath("get_file_attribute"), nil).(string)
	// "get file attribute failed: field does not exist"
	if !isStr || strings.Contains(uri, "failed:") || strings.Contains(uri, "field does not exist") {
		return ""
	}
	return uri
}

func setFieldInner(key string, val interface{}, name string, typ string, attrs map[string]interface{}, ctx context.Context, d *schema.ResourceData, meta interface{}, doDiff bool, isCreate bool, forcedChangeKeys map[string]bool, forcedChangeVals map[string]interface{}) (bool, diag.Diagnostics) {
	compoundRegex, isStr := GetNestedValueOrDefault(attrs, ToKeyPath(key+".compound_in"), nil).(string)
	if isStr {
		curMap := ExtractRegexToMap(CastToString(val), compoundRegex)
		appendActionLog(fmt.Sprintf("CompoundSet: %s: '%s'.'%s' map(%v) from (( %v ))\n", typ, name, key, curMap, val))

		unchanged := map[string]bool{}
		if doDiff {
			old, _ := d.GetChange(key)
			oldMap := ExtractRegexToMap(CastToString(old), compoundRegex)
			for k, v := range oldMap {
				nu, exists := curMap[k]
				if exists && v == nu {
					unchanged[k] = true
				}
			}
		}

		for k, v := range curMap {
			_, skip := unchanged[k]
			if skip {
				continue
			}
			result := setFieldViaOp(typ, attrs, name, k, v)
			if result != nil {
				return false, result
			}
		}
		return true, nil
	}

	result := diag.Diagnostics(nil)
	if forcedChangeKeys[key] {
		result = setFieldViaOp(typ, attrs, name, key, forcedChangeVals[key])
	} else {
		result = setFieldViaOp(typ, attrs, name, key, val)

		// on failure, if field is deprecated and renamed, try the new name
		deprecatedFor := GetNestedValueOrDefault(attrs, ToKeyPath(key+".deprecated_for"), "").(string)
		if deprecatedFor != "" && result != nil {
			appendActionLog(fmt.Sprintf("Set deprecated/renamed field : %s: '%s'.'%s'->'%s'  val:'%v'\n", typ, name, key, deprecatedFor, val))
			result = setFieldViaOp(typ, attrs, name, deprecatedFor, val)
		}
	}
	if result != nil {
		return false, result
	}
	return true, nil
}

func maybeArrayLen(val interface{}) int {
	if val == nil {
		return 0
	}
	arr := reflect.ValueOf(val)
	return arr.Len()
}

func shouldSkipSetField(key string, val interface{}, name string, typ string, attrs map[string]interface{}, ctx context.Context, d *schema.ResourceData, meta interface{}, doDiff bool, isCreate bool, forcedChangeKeys map[string]bool, forcedChangeVals map[string]interface{}, backendVersion VersionRecord) (bool, diag.Diagnostics) {
	skip := GetNestedValueOrDefault(attrs, ToKeyPath(key+".skip"), false).(bool)
	if skip {
		appendActionLog(fmt.Sprintf("Set (skipping explicit): %s: '%s'.'%s'\n", typ, name, key))
		return true, nil
	}

	internal := GetNestedValueOrDefault(attrs, ToKeyPath(key+".internal"), false).(bool)
	if internal {
		appendActionLog(fmt.Sprintf("Set (skipping internal): %s: '%s'.'%s'\n", typ, name, key))
		return true, nil
	}
	proxy := GetNestedValueOrDefault(attrs, ToKeyPath(key+".proxy"), "").(string)
	if proxy != "" {
		appendActionLog(fmt.Sprintf("Set (skipping proxy): %s: '%s'.'%s'\n", typ, name, key))
		return true, nil
	}

	attrTyp := GetNestedValueOrDefault(attrs, ToKeyPath(key+".type"), "string").(string)
	min_ver := GetNestedValueOrDefault(attrs, ToKeyPath(key+".min_ver"), "").(string)
	if min_ver != "" {
		minVer := ParseVersionString(min_ver)
		// XXX check minVer.Error and complain about version string
		gtlteq, valid := CompareVersionRecords(backendVersion, minVer)
		if valid && gtlteq < 0 {
			// NOTE: see below for errata on GetOk().exists
			val, exists := d.GetOk(key)
			defowlt := GetNestedValueOrDefault(attrs, ToKeyPath(key+".default"), nil)
			if defowlt == nil {
				defowlt = AttrValueDefault(attrTyp)
			}
			appendActionLog(fmt.Sprintf("Set (checking min_ver): %s: '%s'.'%s' exists(%v) val(%v : %T) default(%v : %T) ver(%v) backend_ver(%v)\n", typ, name, key, exists, val, val, defowlt, defowlt, min_ver, backendVersion.Version))
			// NOTE: because of the bug in GetOk(), we can't know for sure if the value is set in the TF HCL
			//   e.g. value=<unset>, default=true -> exists==true
			//        value=false,   default=true -> exists==false
			// So, be conservative, and only complain if it's different than the default:
			//
			// special handling for string-array types, since in golang, two empty arrays are not equal
			// (also an emtpy terraform-created 'val' may be []interface{} instead of []string)
			isEmptyArray := false
			if (attrTyp == "string_set" || attrTyp == "string[]") && maybeArrayLen(val) == 0 && maybeArrayLen(defowlt) == 0 {
				isEmptyArray = true
			}
			appendActionLog(fmt.Sprintf("Set (checking min_ver, isEmptyArray: %v): %s: '%s'.'%s' exists(%v) val(%+v -- %T) default(%+v -- %T) ver(%v) backend_ver(%v)\n", isEmptyArray, typ, name, key, exists, val, val, defowlt, defowlt, min_ver, backendVersion.Version))
			if val != nil && val != defowlt && !isEmptyArray {
				// XXX error or warning? (Hashi plugin SDK v2 doesn't seem to support warnings)
				//diags.AddWarning("Below minimum version.", fmt.Sprintf("Field %s.%s requires minimum version %s, skipping...", name, key, min_ver))
				diags := diag.Errorf("Field '%s.%s' requires minimum version '%s', but backend is '%s'", name, key, min_ver, backendVersion.Version)
				return false, diags
			}
			appendActionLog(fmt.Sprintf("Set (skipping): %s: '%s'.'%s' exists(%v) val(%v) default(%v) ver(%v) backend_ver(%v)\n", typ, name, key, exists, val, defowlt, min_ver, backendVersion.Version))
			return true, nil
		}
	}

	max_ver := GetNestedValueOrDefault(attrs, ToKeyPath(key+".max_ver"), "").(string)
	if max_ver != "" {
		maxVersion := ParseVersionString(max_ver)
		gtlteq, valid := CompareVersionRecords(backendVersion, maxVersion)

		if valid && gtlteq >= 0 {
			appendActionLog(fmt.Sprintf("Set (skipping): %s: '%s'.'%s' ver(%v) backend_ver(%v)\n", typ, name, key, max_ver, backendVersion.Version))
			return true, nil
		}
	}

	return false, nil
}

func createUpdateSystemSettingsCommand(systemSettings map[string]interface{}) string {
	var builder strings.Builder
	builder.WriteString("update_configuration(")

	first := true
	for key, value := range systemSettings {
		if !first {
			builder.WriteString(", ")
		}
		first = false

		builder.WriteString(key)
		builder.WriteString("=")

		switch v := value.(type) {
		case string:
			builder.WriteString(fmt.Sprintf("\"%s\"", v))
		case int:
			builder.WriteString(strconv.Itoa(v))
		case bool:
			builder.WriteString(strconv.FormatBool(v))
		case []interface{}:
			encodedList, _ := json.Marshal(v)

			builder.WriteString(string(encodedList))

		default:
			builder.WriteString(fmt.Sprintf("\"%v\"", v))
		}
	}

	builder.WriteString(")")

	return builder.String()
}

func updateSystemSettings(attrs map[string]interface{}, objectDef map[string]interface{}, ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	settingsToUpdate := make(map[string]interface{})

	settingsToUpdate["configuration_name"] = "system_settings"

	skipSettings := map[string]bool{
		"notebook_ad_hoc_approval_request_enabled":      true,
		"notebook_approval_request_expiry_time":         true,
		"notebook_run_approval_expiry_time":             true,
		"parallel_notebook_runs_fired_by_time_triggers": true,
		"runbook_ad_hoc_approval_request_enabled":       true,
		"runbook_approval_request_expiry_time":          true,
		"run_approval_expiry_time":                      true,
		"parallel_runs_fired_by_time_triggers":          true,
	}
	preferredSettings := map[string]string{
		"runbook_ad_hoc_approval_request_enabled": "notebook_ad_hoc_approval_request_enabled",
		"runbook_approval_request_expiry_time":    "notebook_approval_request_expiry_time",
		"run_approval_expiry_time":                "notebook_run_approval_expiry_time",
		"parallel_runs_fired_by_time_triggers":    "parallel_notebook_runs_fired_by_time_triggers",
	}

	for key, _ := range attrs {
		if skipSettings[key] {
			continue
		}
		val, _ := d.GetOk(key)

		defowlt := GetNestedValueOrDefault(attrs, ToKeyPath(key+".default"), nil)
		attrTyp := GetNestedValueOrDefault(attrs, ToKeyPath(key+".type"), "string")

		switch attrTyp {
		case "int":
			if d.HasChange(key) || val.(int) != int(defowlt.(float64)) {
				settingsToUpdate[key] = val
			}
		default:
			if d.HasChange(key) || val != defowlt {
				settingsToUpdate[key] = val
			}
		}
	}

	for preferredSetting, deprecatedSetting := range preferredSettings {

		_, preferredSettingNewValue := d.GetChange(preferredSetting)
		_, deprecatedSettingNewValue := d.GetChange(deprecatedSetting)

		defowlt := GetNestedValueOrDefault(attrs, ToKeyPath(preferredSetting+".default"), nil)
		attrTyp := GetNestedValueOrDefault(attrs, ToKeyPath(preferredSetting+".type"), "string")

		if d.HasChange(preferredSetting) {
			settingsToUpdate[preferredSetting] = preferredSettingNewValue
			continue
		}
		if d.HasChange(deprecatedSetting) {
			settingsToUpdate[deprecatedSetting] = deprecatedSettingNewValue
			continue
		}
		switch attrTyp {
		case "int":

			if preferredSettingNewValue.(int) != int(defowlt.(float64)) {
				settingsToUpdate[preferredSetting] = preferredSettingNewValue
				continue
			}
			if deprecatedSettingNewValue.(int) != int(defowlt.(float64)) {
				settingsToUpdate[deprecatedSetting] = deprecatedSettingNewValue
				continue
			}
		default:
			if preferredSettingNewValue != defowlt {
				settingsToUpdate[preferredSetting] = preferredSettingNewValue
				continue
			}

			if deprecatedSettingNewValue != defowlt {
				settingsToUpdate[deprecatedSetting] = deprecatedSettingNewValue
				continue
			}
		}
	}

	op := createUpdateSystemSettingsCommand(settingsToUpdate)

	appendActionLog(fmt.Sprintf("Updating system settings statement... '%s'\n", op))
	result, err := runOpCommand(op, true)
	if err != nil {
		appendActionLog(fmt.Sprintf("Failed to update system settings: %s\n", err.Error()))

		return diag.Errorf("Failed to update system settings: %s\n", err.Error())
	}
	err = CheckUpdateResult(result)
	if err != nil {
		return diag.Errorf("Failed to update system settings: %s\n", err.Error())
	}

	return nil
}

func ResourceObjectSetFields(typ string, attrs map[string]interface{}, objectDef map[string]interface{}, ctx context.Context, d *schema.ResourceData, meta interface{}, doDiff bool, isCreate bool) diag.Diagnostics {
	var diags diag.Diagnostics
	name := d.Get("name").(string)
	// valid-variable-name check (and non-null)
	appendActionLog(fmt.Sprintf("RESOURCE TYPE IS: %s (ResourceObjectSetFields)\n", typ))

	specialSkipFields := map[string]bool{}
	if typ == "notebook" || typ == "runbook" {
		if notebookIsInline(typ, attrs, objectDef, ctx, d, meta) {
			appendActionLog(fmt.Sprintf("Setting %s:%v :: IS_INLINE\n", typ, name))
			// TODO move this to the json-config
			//specialSkipFields["cells"] = true
			//specialSkipFields["params"] = true
			//specialSkipFields["external_params"] = true
			//specialSkipFields["enabled"] = true

			specialSkipFields["data"] = true
		} else {
			appendActionLog(fmt.Sprintf("Setting %s:%v :: NOT_INLINE\n", typ, name))
			//specialSkipFields["data"] = true

			specialSkipFields["cells"] = true
			specialSkipFields["params"] = true
			specialSkipFields["external_params"] = true
			specialSkipFields["enabled"] = true
		}
	}

	needVersion := false
	writeEnable := false
	enableVal := false
	anyChange := false
	// fields that have to be explicitly set (e.g. notebook fields both in JSON and explicit TF)
	forcedUpdate := map[string]bool{}
	// computed file properties
	forcedChangeKeys := map[string]bool{}
	forcedChangeVals := map[string]interface{}{}

	for key, _ := range attrs {
		proxy := GetNestedValueOrDefault(attrs, ToKeyPath(key+".proxy"), "").(string)
		if proxy != "" {
			proxyKeys := strings.Split(proxy, ",")
			for _, k := range proxyKeys {
				forcedChangeKeys[k] = true
			}
		}
		min_ver := GetNestedValueOrDefault(attrs, ToKeyPath(key+".min_ver"), "").(string)
		if min_ver != "" {
			needVersion = true
		}

		max_ver := GetNestedValueOrDefault(attrs, ToKeyPath(key+".max_ver"), "").(string)
		if max_ver != "" {
			needVersion = true
		}
	}

	var backendVersion VersionRecord
	backendVersion.Valid = false
	if needVersion {
		backendVersion = GetBackendVersionInfoStruct()
	}

	if typ == "file" {
		var err error
		var base64Data string
		var md5sum string
		var fileSize int64

		infileParam, fileParamExists := d.GetOk("input_file")
		contentParam, contentParamExists := d.GetOk("inline_data")

		infile := infileParam.(string)
		content := []byte(contentParam.(string))
		infileLocal := infile

		// Chek if the source is remote
		if strings.HasPrefix(infile, "http:") || strings.HasPrefix(infile, "https://") {
			tmpFileName, err := DownloadFileHttpsToTemp(infile, "")
			if err != nil {
				diags = diag.Errorf("Failed to read remote file object %s: %s", infile, err)
				return diags
			}
			infileLocal = tmpFileName
			defer os.Remove(tmpFileName)
		}

		// Check if the file destination is inline or to a remote store (e.g. S3 / GCS / etc.)
		uri := getRemoteFileAttr(name, "uri")
		fileIsRemote := true
		if uri == "" {
			fileIsRemote = false
		}

		if fileParamExists {
			err, md5sum, fileSize = FileMd5AndSize(infileLocal)
			if !fileIsRemote {
				content, err = ioutil.ReadFile(infileLocal)
				if err != nil {
					diags = diag.Errorf("Failed to read file object %s (%s): %s", infile, infileLocal, err)
					return diags
				}
			}
		} else if contentParamExists {
			md5sum, fileSize = ContentMd5AndSize(content)
		} else {
			diags = diag.Errorf("Must specify input_file or inline_data")
			return diags
		}

		if fileIsRemote {
			base64Data = fmt.Sprintf(":%s", uri)
		} else {
			base64Data = CompressedBase64(content)
		}

		appendActionLog(fmt.Sprintf("file_length is %d (%v)\n", int(fileSize), fileSize))
		if forcedChangeKeys["file_data"] {
			forcedChangeVals["file_length"] = int(fileSize)
			forcedChangeVals["checksum"] = md5sum
			forcedChangeVals["file_data"] = base64Data
		}
		d.Set("file_length", int(fileSize))
		d.Set("checksum", md5sum)
		d.Set("file_data", base64Data)
		if fileIsRemote {
			presignedUrl := getRemoteFileAttr(name, "presigned_put")
			if presignedUrl == "" {
				diags = diag.Errorf("Failed to get presigned url for file object %s", name)
				return diags
			}
			var err error
			if contentParamExists {
				err = UploadFileHttpsFromString(string(content), presignedUrl, "")
				if err == nil {
					d.Set("inline_data", string(content))
				}
			} else {
				err = UploadFileHttps(infileLocal, presignedUrl, "")
			}
			if err != nil {
				diags = diag.Errorf("Failed to upload to presigned url for file object %s -- %s", name, err.Error())
				return diags
			}
		}
	}

	// Have to explicitly set "data" first, as it overrides some other attributes (e.g. "approvers")
	if typ == "notebook" || typ == "runbook" {
		var runbookData interface{}
		var err error

		key := "cells"
		cells, exists := d.GetOk(key)
		//appendActionLog(fmt.Sprintf("cell data value exists:%v, hasChange():%v, value: %v\n", exists, d.HasChange(key), cells))
		// NOTE: Terraform reports !exists when a value is explicitly supplied, but matches the 'default'
		//if exists || d.HasChange(key) {
		if notebookIsInline(typ, attrs, objectDef, ctx, d, meta) {
			if exists {
				runbookData, err = buildRunbookDataObject(d, CastToObject(cells))
				appendActionLog(fmt.Sprintf("buildRunbookDataObject input: [[[ %v ]]]\n", runbookData))
				appendActionLog(fmt.Sprintf("buildRunbookDataObject output: [[[ %v ]]]\n", runbookData))
				if err != nil {
					diags = diag.Errorf("Failed to build runbook data object: %s", err)
					return diags
				}
			}
		} else {
			key = "data"
			val, exists := d.GetOk(key)
			// NOTE: Terraform reports !exists when a value is explicitly supplied, but matches the 'default'
			if exists || d.HasChange(key) {
				runbookData = val
			}
		}

		changed, diags := setFieldInner("data", runbookData, name, typ, attrs, ctx, d, meta, doDiff, isCreate, forcedChangeKeys, forcedChangeVals)
		if diags != nil {
			return diags
		}
		if changed {
			anyChange = true
		}

		forced, hasForced := GetNestedValueOrDefault(attrs, ToKeyPath(key+".force_set"), false).([]interface{})
		if hasForced {
			for _, k := range forced {
				forcedUpdate[CastToString(k)] = true
			}
		}
	}

	orderedAttrs := []string{}
	skipKeys := map[string]bool{}
	if typ == "notebook" || typ == "runbook" {
		skipKeys["cells"] = true           // aggregated into the `data` field
		skipKeys["params"] = true          // aggregated into the `data` field
		skipKeys["external_params"] = true // aggregated into the `data` field
		//skipKeys["enabled"] = true         // aggregated into the `data` field
		skipKeys["data"] = true
		skipKeys["approvers"] = true
		skipKeys["allowed_entities"] = true
		if notebookIsInline(typ, attrs, objectDef, ctx, d, meta) {
			skipKeys["data"] = true
		} else {
			skipKeys["enabled"] = true // aggregated into the `data` field
		}
	}

	if typ == "principal" {
		orderedAttrs = append(orderedAttrs, "idp_name")
	}

	for key, _ := range attrs {
		forceUpdate := GetNestedValueOrDefault(attrs, ToKeyPath(key+".force_update"), false).(bool)
		if forceUpdate {
			forcedUpdate[CastToString(key)] = forceUpdate
		}

		if typ == "principal" && key == "idp_name" {
			continue
		}

		if skipKeys[key] != true {
			orderedAttrs = append(orderedAttrs, key)
		} else {
			appendActionLog(fmt.Sprintf("Notebook skipping key: %s\n", key))
		}
	}

	if typ == "notebook" || typ == "runbook" {
		// XXX: work around backend issue with data-dependent ordering
		aVal, _ := d.Get("allowed_entities").([]interface{})
		//appendActionLog(fmt.Sprintf("Notebook allowed_entities has len: %v\n", len(aVal)))
		if len(aVal) > 0 {
			orderedAttrs = append(orderedAttrs, "allowed_entities")
			orderedAttrs = append(orderedAttrs, "approvers")
		} else {
			orderedAttrs = append(orderedAttrs, "approvers")
			orderedAttrs = append(orderedAttrs, "allowed_entities")
		}
	}

	for _, key := range orderedAttrs {
		if specialSkipFields[key] {
			val := "nil"
			appendActionLog(fmt.Sprintf("Skipping set (special-skip) %s field: '%s'.'%s' :: %+v\n", typ, name, key, val))
			continue
		}
		// NOTE: GetOk() has bugs: it checks vs 0/false/"" instead of presence of an explicit value, or even equality to the default
		val, exists := d.GetOk(key)

		skip, diags := shouldSkipSetField(key, val, name, typ, attrs, ctx, d, meta, doDiff, isCreate, forcedChangeKeys, forcedChangeVals, backendVersion)
		if diags != nil {
			return diags
		}
		if skip {
			continue
		}
		if (typ == "notebook" || typ == "runbook") && key == "data" {
			// set first above, skip here
			continue
		}

		forceSet := false
		// CS-336 workaround: Force explicit set of action_statement/alarm_statement to patch quoting issue
		_, botEnvDefined := os.LookupEnv("BOT_SKIP_PATCH")
		isPrimary := GetNestedValueOrDefault(attrs, ToKeyPath(key+".primary"), false).(bool)
		if isCreate && isPrimary && typ == "bot" {
			if botEnvDefined {
				appendActionLog(fmt.Sprintf("Bot skipping post-ctor set: %s: '%s'.'%s' HasChange(%v)\n", typ, name, key, d.HasChange(key)))
				// primary value is set on creation, and redundant set currently triggers an issue with bots
				continue
			} else {
				appendActionLog(fmt.Sprintf("Bot running post-ctor set: %s: '%s'.'%s'  HasChange(%v)\n", typ, name, key, d.HasChange(key)))
				forceSet = true
			}
		}

		if typ == "principal" && key == "idp_name" {
			forceSet = true
		}

		// NOTE: Terraform reports !exists when a value is explicitly supplied, but matches the 'default'
		defowlt := GetNestedValueOrDefault(attrs, ToKeyPath(key+".default"), nil)

		if !exists && !d.HasChange(key) && !forceSet && !forcedChangeKeys[key] && !forcedUpdate[key] {
			appendActionLog(fmt.Sprintf("FieldDoesNotExist: %s: '%s'.'%s' val(%v) HasChange(%v), forceSet(%v) isCreate(%v) default(%v)\n", typ, name, key, val, d.HasChange(key), forceSet, isCreate, defowlt))
			// Handle GetOk() bug...
			if isCreate {
				if defowlt == nil || val == defowlt {
					continue
				}
			} else {
				continue
			}
		}

		// Because OpLang auto-toggles some objects to "disabled" on *any* property change,
		// we have to restore the value as needed.
		if key == "enabled" {
			enableVal, _ = CastToBoolMaybe(val)
			if d.HasChange(key) || !doDiff {
				writeEnable = true
			}
			appendActionLog(fmt.Sprintf("CheckEnableState: %s: '%s' write(%v) val(%v) change(%v) hasChange:(%v) doDiff(%v)\n", typ, name, writeEnable, enableVal, anyChange, d.HasChange(key), doDiff))
			continue
		}
		if doDiff && !d.HasChange(key) && !forcedChangeKeys[key] && !forcedUpdate[key] {
			continue
		}

		attrsMergedWithAlias := attrs
		aliasKey, _ := GetNestedValueOrDefault(objectDef, ToKeyPath("internal.alias.key"), "").(string)
		aliasKeyVal := ""
		aliasMap := map[string]interface{}{}
		if aliasKey != "" {
			aliasKeyVal = d.Get(aliasKey).(string)
			aliasMap = GetNestedValueOrDefault(objectDef, ToKeyPath("internal.alias.map."+aliasKeyVal), map[string]interface{}{}).(map[string]interface{})
			if aliasMap != nil {
				// Create a merged map instead of replacing
				attrsMergedWithAlias = make(map[string]interface{})
				// Copy base attributes
				for k, v := range attrs {
					attrsMergedWithAlias[k] = v
				}
				// Override with alias-specific attributes
				for k, v := range aliasMap {
					attrsMergedWithAlias[k] = v
				}
			}
		}

		changed, diags := setFieldInner(key, val, name, typ, attrsMergedWithAlias, ctx, d, meta, doDiff, isCreate, forcedChangeKeys, forcedChangeVals)
		if diags != nil {
			return diags
		}
		if changed {
			anyChange = true
		}
	}

	appendActionLog(fmt.Sprintf("EnableState: %s: '%s' write(%v) val(%v) anyChange(%v)\n", typ, name, writeEnable, enableVal, anyChange))
	// Enabled is automatically toggled to "false" by oplang on any other attribute change.
	// So, it requires special handling.
	if writeEnable || (enableVal && anyChange) {
		act := "enable"
		if !enableVal {
			act = "disable"
		}
		op := fmt.Sprintf("%s %s", act, name)
		appendActionLog(fmt.Sprintf("EnableState: %s: '%s' Op:'%s'\n", typ, name, op))
		result, err := runOpCommand(op, true)
		if err != nil {
			diags = diag.Errorf("Failed to %s (1) %s: %s", act, typ, err.Error())
			return diags
		}
		err = CheckUpdateResult(result)
		if err != nil {
			diags = diag.Errorf("Failed to %s (2) %s: %s", act, typ, err.Error())
			return diags
		}
	}
	return nil
}

func notebookIsInline(typ string, attrs map[string]interface{}, objectDef map[string]interface{}, ctx context.Context, d *schema.ResourceData, meta interface{}) bool {
	key := "cells"
	cells, cellsExists := d.GetOk(key)
	appendActionLog(fmt.Sprintf("Runbook 'cells' value... exists:%v, hasChange():%v, value(%T): %v\n", cellsExists, d.HasChange(key), cells, cells))
	key = "data"
	data, dataExists := d.GetOk(key)
	appendActionLog(fmt.Sprintf("Runbook 'data' value... exists:%v, hasChange():%v, value(%T): %v\n", dataExists, d.HasChange(key), data, data))

	// NOTE: Terraform reports !exists when a value is explicitly supplied, but matches the 'default'
	// HasChange() has some similar deficiencies (especially after initial apply)...
	if cellsExists && cells != nil && cells != "" {
		//appendActionLog(fmt.Sprintf("InlineCheck: cellsExists(%v) isNill(%v) isEmptyStr(%v) :: value= %+v\n", cellsExists, (cells == nil), (cells == ""), cells))
		return true
	}
	if dataExists && data != nil && data != "" {
		//appendActionLog(fmt.Sprintf("InlineCheck: dataExists(%v) isNill(%v) isEmptyStr(%v) :: value= %+v\n", dataExists, (data == nil), (data == ""), data))
		return false
	}
	appendActionLog(fmt.Sprintf("InlineCheck: DEFAULT\n"))
	return false
}

func ResourceObjectCreate(typ string, primary string, attrs map[string]interface{}, objectDef map[string]interface{}) func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		// use the meta value to retrieve your client from the provider configure method
		// client := meta.(*apiClient)

		var diags diag.Diagnostics
		name := d.Get("name").(string)
		primaryVal := d.Get(primary)
		idFromAPI := name
		appendActionLog(fmt.Sprintf("Creating %s: '%s' (%v) :: %+v\n", typ, idFromAPI, name, d))

		singletonName, _ := GetNestedValueOrDefault(objectDef, ToKeyPath("internal.singleton"), "").(string)
		appendActionLog(fmt.Sprintf("Creating %s: '%s' (%v) :: %+v -- singletonName: '%v'\n", typ, idFromAPI, name, d, singletonName))
		if singletonName != "" {
			if name != singletonName {
				diags = diag.Errorf("Invalid name for %s singleton object '%s' vs required name '%s'", typ, name, singletonName)
				return diags
			}
		}
		// TODO return early if "no_create"
		isNoCreate, _ := GetNestedValueOrDefault(objectDef, ToKeyPath("internal.no_create"), false).(bool)
		if isNoCreate {
			// once the object is ok, set the ID to tell terraform it's valid...
			d.SetId(name)
			// update the data in terraform
			return ResourceObjectUpdate(typ, attrs, objectDef)(ctx, d, meta)
		}

		primaryValStr := attrValueString(typ, primary, primaryVal, attrs)
		if typ == "notebook" || typ == "runbook" {
			if primaryValStr == "" || primaryValStr == "\"\"" {
				// base64 encoded "{}", as notebooks don't seem to allow creation with an empty string anymore
				// however, we need to create the notebook "empty", when there's no file-data provided
				primaryValStr = "\"e30K\""
				//appendActionLog(fmt.Sprintf("Create setting notebook %s primary to '%s'\n", name, primaryValStr))
			}
		}
		//appendActionLog(fmt.Sprintf("primaryValStr is ((( %+v )))\n", primaryValStr))
		//op := fmt.Sprintf("%s %s = \"%s\"", typ, name, primaryVal)
		op := ""
		if typ == "nvault_secret" {
			integrationName := d.Get("integration_name")
			vaultSecretPath := d.Get("vault_secret_path")
			vaultSecretKey := d.Get("vault_secret_key")

			op = fmt.Sprintf("%s %s = nvault_config(\"%s\", \"%s\", \"%s\")", typ, name, integrationName, vaultSecretPath, vaultSecretKey)
		} else {
			op = fmt.Sprintf("%s %s = %s", typ, name, primaryValStr)
		}
		//if typ == "bot" {
		//	// special handling for BOT creation statement "bot <name>=
		//	action := d.Get("action_statement").(string)
		//	alarm := d.Get("alarm_statement").(string)
		//	op = fmt.Sprintf("%s %s = if %s then %s fi", typ, name, alarm, action)
		//}
		result, err := runOpCommand(op, true)
		if err != nil {
			// TODO check if already exists
			diags = diag.Errorf("Failed to create (1) %s: %s", typ, err.Error())
			return diags
		}
		err = CheckUpdateResult(result)
		if err != nil {
			diags = diag.Errorf("Failed to create (2) %s: %s", typ, err.Error())
			return diags
		}

		diags = ResourceObjectSetFields(typ, attrs, objectDef, ctx, d, meta, false, true)
		if diags != nil {
			// delete incomplete object
			ResourceObjectDelete(typ, objectDef)(ctx, d, meta)
			return diags
		}

		// once the object is ok, set the ID to tell terraform it's valid...
		d.SetId(name)
		// update the data in terraform
		return ResourceObjectRead(typ, attrs, objectDef)(ctx, d, meta)
	}
}

// returns skip, value, diagnostics
func ResourceObjectReadSingleAttr(name string, typ string, key string, attrs map[string]interface{}, record map[string]interface{}, stepsJs map[string]interface{}, d *schema.ResourceData, alias string, aliasMap map[string]interface{}) (bool, interface{}, diag.Diagnostics) {
	var val interface{}
	attr := GetNestedValueOrDefault(attrs, ToKeyPath(key), map[string]interface{}{})
	if alias != "" {
		aliasedAttr := GetNestedValueOrDefault(aliasMap, ToKeyPath(key), nil)
		if aliasedAttr != nil {
			attr = aliasedAttr
		}
	}

	if strings.HasPrefix(key, "#") {
		// skip commented fields
		return true, nil, nil
	}

	internal := GetNestedValueOrDefault(attr, ToKeyPath("internal"), false).(bool)
	if internal {
		// skip internal fields
		return true, nil, nil
	}

	compoundValue, isStr := GetNestedValueOrDefault(attr, ToKeyPath("compound_out"), nil).(string)
	if isStr {
		fullVal := compoundValue
		re := regexp.MustCompile(`\$\{\w\w*\}`)
		for expr := re.FindString(fullVal); expr != ""; expr = re.FindString(fullVal) {
			l := len(expr)
			varName := expr[2 : l-1]
			valStr := CastToString(GetNestedValueOrDefault(record, ToKeyPath("attributes."+varName), ""))
			fullVal = strings.Replace(fullVal, expr, valStr, -1)
		}
		val = fullVal
	} else {
		stepPath, isStr := GetNestedValueOrDefault(attr, ToKeyPath("step"), nil).(string)
		if isStr {
			if stepPath == "." {
				val = stepsJs
			} else {
				val = GetNestedValueOrDefault(stepsJs, ToKeyPath(stepPath), nil)
			}

			// special handling (notebooks)... field is base64 outgoing, and json incoming
			attrTyp := GetNestedValueOrDefault(attr, ToKeyPath("type"), "string").(string)
			if attrTyp == "b64json" {
				// The code below that omits fields/objects will modify 'val', so we make a copy
				val = DeepCopy(val)
				// handle cast-map, as get_notebook_class() returns objects with some string-wrapped sub-fields
				castMap := GetNestedValueOrDefault(attr, ToKeyPath("cast"), map[string]interface{}{}).(map[string]interface{})
				for castPath, castType := range castMap {
					cur := GetNestedValueOrDefault(val, ToKeyPath(castPath), nil)
					if cur != nil {
						// TODO add additional types as needed
						switch castType {
						case "string[]":
							SetNestedValue(val, ToKeyPath(castPath), CastToArray(cur))
						case "string_set":
							SetNestedValue(val, ToKeyPath(castPath), CastToArray(cur))
						case "object":
							SetNestedValue(val, ToKeyPath(castPath), CastToObject(cur))
						}
					}
				}

				// handle omit map, and nested deletions, as get_notebook_class() returns objects with dynamic/temporary fields
				omitMap := GetNestedValueOrDefault(attr, ToKeyPath("omit"), map[string]interface{}{}).(map[string]interface{})
				// "." has to be last, or it will wipe out other objects
				omitPaths := []string{}
				hasDot := false
				for omitPath, _ := range omitMap {
					if omitPath != "." {
						omitPaths = append(omitPaths, omitPath)
					} else {
						hasDot = true
					}
				}
				if hasDot {
					omitPaths = append(omitPaths, ".")
				}
				for _, omitPath := range omitPaths {
					omitTag := omitMap[omitPath]
					appendActionLog(fmt.Sprintf("Omit path:'%+v' tag: '%+v'\n", omitPath, omitTag))
					var cur interface{}
					if omitPath == "." {
						cur = val
					} else {
						cur = GetNestedValueOrDefault(val, ToKeyPath(omitPath), nil)
					}
					omitTagStr, isStr := omitTag.(string)
					if !isStr {
						// skip omit fields
						return true, nil, nil
					}
					omitList, isList := GetNestedValueOrDefault(stepsJs, ToKeyPath(omitTagStr), []interface{}{}).([]interface{})
					//appendActionLog(fmt.Sprintf("Omit-list path:'%+v' tag: '%+v' list:'%+v'\n", omitPath, omitTag, omitList))
					if cur != nil && isList {
						if (typ == "notebook" || typ == "runbook") && omitPath == "." {
							// NOTE: The top-level object returned by get_notebook_class contains most/all of the object attributes.
							// So remove them from the inner object
							for akey, _ := range attrs {
								if akey != "cells" && akey != "params" && akey != "external_params" {
									omitList = append(omitList, akey)
								}
							}
							omitList = append(omitList, "enabled")
						}
						switch cur.(type) {
						case map[string]interface{}:
							OmitJsonObjectFields(cur.(map[string]interface{}), omitList)
						case []interface{}:
							curArr := cur.([]interface{})
							OmitJsonArrayFields(&curArr, omitList)
						}
						if omitPath == "." {
							val = cur
						} else {
							SetNestedValue(val, ToKeyPath(omitPath), cur)
						}
					}
				}

				// handle dynamic parameters (eg. datadog external params)
				omitMap = GetNestedValueOrDefault(attr, ToKeyPath("omit_items"), map[string]interface{}{}).(map[string]interface{})
				for omitPath, omitTag := range omitMap {
					cur, isList := GetNestedValueOrDefault(val, ToKeyPath(omitPath), nil).([]interface{})
					if cur == nil || !isList {
						// skip ...
						return true, nil, nil
					}
					omitTagStr, isStr := omitTag.(string)
					if !isStr {
						// skip omit fields
						return true, nil, nil
					}
					omitList, isList := GetNestedValueOrDefault(stepsJs, ToKeyPath(omitTagStr), []interface{}{}).([]interface{})
					if !isList {
						// skip omit fields
						return true, nil, nil
					}
					OmitJsonArrayItems(&cur, omitList)
					SetNestedValue(val, ToKeyPath(omitPath), cur)
				}

				b, err := json.Marshal(val)
				if err != nil {
					diags := diag.Errorf("Failed to marshall JSON %s:%s '%s'", typ, key, name)
					return false, nil, diags
				}
				//val = base64.URLEncoding.EncodeToString(b)
				val = string(b)
			}
		} else {
			val = GetNestedValueOrDefault(record, ToKeyPath("attributes."+key), nil)
		}
	}
	return false, val, nil
}

func SetSingleAttrFromRead(typ string, name string, key string, val interface{}, attrs map[string]interface{}, ctx context.Context, d *schema.ResourceData, meta interface{}) {
	appendActionLog(fmt.Sprintf("Reading (updating local state) %s field: '%s'.'%s' ::(%T) %+v\n", typ, name, key, val, val))
	attrTyp := GetNestedValueOrDefault(attrs, ToKeyPath(key+".type"), "string").(string)
	switch attrTyp {
	case "float":
		d.Set(key, float64(CastToNumber(val)))
	case "int":
		d.Set(key, CastToInt(val))
	case "unsigned":
		d.Set(key, CastToInt(val))
	case "bool":
		d.Set(key, CastToBool(val))
	case "intbool":
		d.Set(key, CastToBool(val))
	case "string[]":
		d.Set(key, CastToArray(val))
	case "string_set":
		d.Set(key, CastToArray(val))
	case "string":
		d.Set(key, CastToString(val))
	case "command":
		d.Set(key, CastToString(val))
	case "label":
		d.Set(key, CastToString(val))
	case "time_s":
		d.Set(key, CastToString(val)+"s")
	default:
		d.Set(key, val)
	}
}

func ResourceObjectRead(typ string, attrs map[string]interface{}, objectDef map[string]interface{}) func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		// use the meta value to retrieve your client from the provider configure method
		// client := meta.(*apiClient)

		var diags diag.Diagnostics
		name := d.Get("name").(string)
		if name == "" {
			// fallback for 'terraform import'
			name = d.Id()
		}
		// valid-variable-name check
		idFromAPI := name
		appendActionLog(fmt.Sprintf("Reading %s: '%s' (%v) :: %+v\n", typ, idFromAPI, name, d))

		specialSkipFields := map[string]bool{}
		if typ == "notebook" || typ == "runbook" {
			if notebookIsInline(typ, attrs, objectDef, ctx, d, meta) {
				appendActionLog(fmt.Sprintf("Reading %s:%v :: IS_INLINE\n", typ, name))
				// TODO move this to the json-config
				//specialSkipFields["cells"] = true
				//specialSkipFields["params"] = true
				//specialSkipFields["external_params"] = true
				//specialSkipFields["enabled"] = true

				specialSkipFields["data"] = true
			} else {
				appendActionLog(fmt.Sprintf("Reading %s:%v :: NOT_INLINE\n", typ, name))
				//specialSkipFields["data"] = true

				specialSkipFields["cells"] = true
				specialSkipFields["params"] = true
				specialSkipFields["external_params"] = true
				specialSkipFields["enabled"] = true
			}
		}

		// return early if "read_single_attr"
		readSingleAttr, _ := GetNestedValueOrDefault(objectDef, ToKeyPath("internal.read_single_attr"), false).(bool)
		if readSingleAttr {
			for key, _ := range attrs {
				if key == "type" || key == "name" {
					continue
				}
				op := fmt.Sprintf("%s.%s", name, key)
				js, err := runOpCommandToJson(op)
				if err != nil {
					diags = diag.Errorf("Failed to read %s - %s.%s: %s", typ, name, key, err.Error())
					return diags
				}

				val := GetNestedValueOrDefault(js, ToKeyPath("get_configuration_attribute"), nil)
				if val != nil {
					SetSingleAttrFromRead(typ, name, key, val, attrs, ctx, d, meta)
				}
			}
			return diags
		}

		op := fmt.Sprintf("list %ss | name = \"%s\"", typ, name)
		js, err := runOpCommandToJson(op)
		if err != nil {
			diags = diag.Errorf("Failed to read %s - %s: %s", typ, name, err.Error())
			return diags
		}

		stepsJs := map[string]interface{}{}

		if typ == "alarm" || typ == "action" || typ == "bot" || typ == "integration" || typ == "notebook" || typ == "runbook" || typ == "time_trigger" || typ == "report_template" || typ == "dashboard" || typ == "nvault_secret" {
			// extract fields from step objects
			op := ""
			if typ == "nvault_secret" {
				op = fmt.Sprintf("get_secret( secret_name = \"%s\" )", name)
			} else {
				op = fmt.Sprintf("get_%s_class( %s_name = \"%s\" )", typ, typ, name)
			}
			extraJs, err := runOpCommandToJson(op)
			if err != nil {
				diags = diag.Errorf("Failed to read %s - %s: %s", typ, name, err.Error())
				return diags
			}
			stepsJs = getNamedObjectFromClassDef(name, typ, extraJs)

			if typ == "integration" {
				// unpack attributes.configuration (integration) which is a string-encoded JSON value
				confStr, hasConfStr := GetNestedValueOrDefault(stepsJs, ToKeyPath("params"), nil).(string)
				if hasConfStr {
					conf := map[string]interface{}{}
					err := json.Unmarshal([]byte(confStr), &conf)
					if err == nil {
						SetNestedValue(stepsJs, ToKeyPath("params_unpack"), conf)
					}
				}
			}
			if typ == "dashboard" {
				confStr, hasConfStr := GetNestedValueOrDefault(stepsJs, ToKeyPath("configuration"), nil).(string)

				if hasConfStr {
					conf := map[string]interface{}{}
					err := json.Unmarshal([]byte(confStr), &conf)

					if err == nil {
						SetNestedValue(stepsJs, ToKeyPath("dashboard_configuration"), conf)
					}
				}
			}
			if typ == "nvault_secret" {
				secretInfo := GetNestedValueOrDefault(stepsJs, ToKeyPath("secret_info"), nil)
				SetNestedValue(stepsJs, ToKeyPath("secret_info"), secretInfo)
			}
		}

		found := false
		record := map[string]interface{}{}
		symbols, isArray := GetNestedValueOrDefault(js, ToKeyPath("list_type.symbol"), []interface{}{}).([]interface{})
		if isArray {
			for _, s := range symbols {
				sName, isStr := GetNestedValueOrDefault(s, ToKeyPath("attributes.name"), "").(string)
				if isStr && name == sName {
					record = s.(map[string]interface{})
					found = true
				}
			}
		}

		if !found {
			diags = diag.Errorf("Failed to find %s '%s'", typ, name)
			return diags
		}

		aliasKey, _ := GetNestedValueOrDefault(objectDef, ToKeyPath("internal.alias.key"), "").(string)
		aliasKeyVal := ""
		aliasMap := map[string]interface{}{}
		if aliasKey != "" {
			_, aliasKeyValIfc, diags := ResourceObjectReadSingleAttr(name, typ, aliasKey, attrs, record, stepsJs, d, "", nil)
			if diags != nil {
				return diags
			}
			aliasKeyVal, _ = aliasKeyValIfc.(string)
			aliasMap = GetNestedValueOrDefault(objectDef, ToKeyPath("internal.alias.map."+aliasKeyVal), map[string]interface{}{}).(map[string]interface{})
		}

		readFirst := map[string]bool{"enable": true, "cells": true, "secret_aware": true}
		attrFirst := []string{}
		attrList := []string{}

		if typ == "notebook" || typ == "runbook" {
			// Normalizing the notebook removes some fields, so we have to read them first
			for key, _ := range attrs {
				if readFirst[key] {
					attrFirst = append(attrFirst, key)
				} else {
					attrList = append(attrList, key)
				}
			}
			attrList = append(attrFirst, attrList...)
		} else {
			for key, _ := range attrs {
				attrList = append(attrList, key)
			}
		}

		//for key, _ := range attrs {
		for _, key := range attrList {
			if specialSkipFields[key] {
				val := "nil"
				appendActionLog(fmt.Sprintf("Skipping read (special-skip) %s field: '%s'.'%s' :: %+v\n", typ, name, key, val))
				continue
			}

			curAlias, _ := GetNestedValueOrDefault(aliasMap, ToKeyPath(key+".alias_out"), "").(string)
			skip, val, diags := ResourceObjectReadSingleAttr(name, typ, key, attrs, record, stepsJs, d, curAlias, aliasMap)

			if diags != nil {
				return diags
			}
			if skip {
				appendActionLog(fmt.Sprintf("Reading (skip) %s field: '%s'.'%s' :: %+v\n", typ, name, key, val))
				continue
			}

			// if there's an obsolete prior name, with a value set, skip
			replaces := GetNestedValueOrDefault(attrs, ToKeyPath(key+".replaces"), "").(string)
			if replaces != "" {
				_, replacesSet := d.GetOk(replaces)
				if replacesSet {
					safeVal := strings.ReplaceAll(fmt.Sprintf("%v", val), "'", "\\'")
					appendActionLog(fmt.Sprintf("Reading deprecated/renamed skipping new (for obsolete) field : %s: '%s'.'%s'->'%s'  '%v'\n", typ, name, key, replaces, safeVal))
					continue
				}
			}

			// on failure, if field is deprecated and renamed and set in HCL, try the new name
			deprecatedFor := GetNestedValueOrDefault(attrs, ToKeyPath(key+".deprecated_for"), "").(string)
			if deprecatedFor != "" && val == nil {
				appendActionLog(fmt.Sprintf("Reading deprecated/renamed field : %s: '%s'.'%s'->'%s'  '%v'\n", typ, name, key, deprecatedFor, val))
				_, isSet := d.GetOk(key)
				if isSet {
					_, val, diags = ResourceObjectReadSingleAttr(name, typ, key, attrs, record, stepsJs, d, curAlias, aliasMap)
				}
			}
			if val == nil {
				if typ == "principal" && key == "idp_name" {
					currentVal, _ := d.GetOk("idp_name")
					if currentVal != nil {
						d.Set("idp_name", currentVal)
					}
					continue
				}

				// not found, so check for a default value and assume that
				defowlt := GetNestedValueOrDefault(attrs, ToKeyPath(key+".default"), nil)
				if defowlt != nil {
					val = defowlt
					appendActionLog(fmt.Sprintf("Reading (default) %s field: '%s'.'%s' :: %+v\n", typ, name, key, val))
					//appendActionLog(fmt.Sprintf("Reading (default) %s field: '%s'.'%s' steps js::     %+v\n", typ, name, key, stepsJs))
				} else {
					// XXX error?
					appendActionLog(fmt.Sprintf("Reading (failed/empty) %s field: '%s'.'%s' :: %+v\n", typ, name, key, val))
					if typ == "file" {
						if key == "input_file" || key == "md5" || key == "inline_data" {
							continue
						}
					}
					// NOTE: If we don't set a value, TF won't do comparisons or show a diff for this field.
					d.Set(key, nil)
					continue
				}
			}

			if key == "cells" && (typ == "notebook" || typ == "runbook") {
				val = GetNestedValueOrDefault(stepsJs, ToKeyPath("cells"), nil)
				// Later cleanup (omit) of fields in 'data' may affect this, so copy...
				val = DeepCopy(val)
				valArr := val.([]interface{})
				NormalizeNotebookCells(&valArr)
				appendActionLog(fmt.Sprintf("Reading (special notebook.cells) %s field: '%s'.'%s' :: %+v\n", typ, name, key, valArr))
				d.Set(key, CastToString(valArr))
				continue
			}

			SetSingleAttrFromRead(typ, name, key, val, attrs, ctx, d, meta)
		}
		return diags
	}
}

func ResourceObjectUpdate(typ string, attrs map[string]interface{}, objectDef map[string]interface{}) func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		// use the meta value to retrieve your client from the provider configure method
		// client := meta.(*apiClient)

		var diags diag.Diagnostics
		name := d.Get("name").(string)
		appendActionLog(fmt.Sprintf("Updated object '%s': '%s' :: %+v\n", typ, name, d))

		if typ == "system_settings" {
			diags = updateSystemSettings(attrs, objectDef, ctx, d, meta)
		} else {
			diags = ResourceObjectSetFields(typ, attrs, objectDef, ctx, d, meta, true, false)
		}
		if diags != nil {
			// TODO delete incomplete object?
			return diags
		}

		// update the data in terraform
		return ResourceObjectRead(typ, attrs, objectDef)(ctx, d, meta)
	}
}

func ResourceObjectDelete(typ string, objectDef map[string]interface{}) func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		// use the meta value to retrieve your client from the provider configure method
		// client := meta.(*apiClient)

		var diags diag.Diagnostics
		name := d.Get("name").(string)
		appendActionLog(fmt.Sprintf("deleting %s: '%s' :: %+v\n", typ, name, d))

		// return early if "no_delete"
		isNoCreate, _ := GetNestedValueOrDefault(objectDef, ToKeyPath("internal.no_delete"), false).(bool)
		if isNoCreate {
			return diags
		}

		op := fmt.Sprintf("delete %s", name)
		result, err := runOpCommand(op, true)
		if err != nil {
			// TODO check already exists
			diags = diag.Errorf("Failed to delete %s: %s", typ, err.Error())
		}
		err = CheckUpdateResult(result)
		if err != nil {
			diags = diag.Errorf("Failed to delete %s: %s", typ, err.Error())
			return diags
		}
		return diags
	}
}
