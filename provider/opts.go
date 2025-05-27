// Copyright 2025 NVIDIA Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	zstd "github.com/klauspost/compress/zstd"
	"github.com/spf13/viper"

	//"io/ioutil"
	prand "math/rand"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type CliOpts struct {
	Execute     bool
	Verbose     bool
	StdIn       bool
	Quiet       bool
	CfgFile     string
	HasAuth     bool
	AuthChanged bool
	Url         string
	Token       string
}

var AuthUrl string
var AuthToken string
var RetryLimit int
var DoDebugLog = false
var GlobalOpts = CliOpts{}

var clientAuth *ClientAuth

var AuthConfig = viper.New()

func GetHomeDir() string {
	user, err := user.Current()
	homeDir := "/"
	if err == nil {
		homeDir = user.HomeDir
	}
	return homeDir
}

func GetDotfilePath() string {
	// TODO patch for windows
	home := GetHomeDir()

	// default to "~/"
	thePath := home
	// check if '~/.<ProviderShortName>/' exists
	shorePath := filepath.Join(home, "."+ProviderShortName)
	if _, err := os.Stat(shorePath); !os.IsNotExist(err) {
		thePath = shorePath
	}
	return thePath
}

func getAuthFilename() string {
	return ".ops_auth.yaml"
}

func getAuthUrls() []string {
	urls := []string{}
	auth := AuthConfig.Get("Auth")
	if auth != nil {
		authArr, isArr := auth.([]interface{})
		if isArr {
			for _, obj := range authArr {
				objMap, isMap := obj.(map[interface{}]interface{})
				if isMap {
					url, uOk := objMap["Url"]
					if uOk {
						urlStr, isStr := url.(string)
						if isStr {
							urls = append(urls, urlStr)
						}
					}
				}
			}
		}
	}
	return urls
}

func LoadAuthConfig(GlobalOpts *CliOpts) bool {
	AuthConfig.SetConfigName(getAuthFilename())
	AuthConfig.SetConfigType("yaml")
	//AuthConfig.AddConfigPath(GetHomeDir())
	AuthConfig.AddConfigPath(GetDotfilePath())
	AuthConfig.ReadInConfig() // ignore errors...

	URL := AuthConfig.GetString("Url")
	TOKEN := AuthConfig.GetString("Token")
	//WriteMsg("*** URL: %s, TOKEN:%.16s\n", URL, TOKEN)
	if URL == "" || TOKEN == "" {
		GlobalOpts.HasAuth = false
	} else {
		GlobalOpts.HasAuth = true
		GlobalOpts.Url = URL
		GlobalOpts.Token = TOKEN
	}
	return GlobalOpts.HasAuth
}

func SetAuth(GlobalOpts *CliOpts, Url string, Token string) {
	AuthUrl = Url
	AuthToken = Token
	// set default
	GlobalOpts.Url = Url
	GlobalOpts.Token = Token
	GlobalOpts.HasAuth = true
	GlobalOpts.AuthChanged = true

	//AddAuthEntry(GlobalOpts, Url, Token, false)
}

func selectAuth(GlobalOpts *CliOpts, toUrl string) bool {
	auth := AuthConfig.Get("Auth")
	if auth == nil {
		WriteMsg("No 'Auth' object in config!\n")
		return false
	}
	authArr, isArr := auth.([]interface{})
	if !isArr {
		WriteMsg("Config 'Auth' object is not an array!\n")
		return false
	}
	for i, obj := range authArr {
		objMap, isMap := obj.(map[interface{}]interface{})
		if !isMap {
			WriteMsg("Config 'Auth' element %d is not an object (%T)!\n", i, obj)
			continue
		}
		urlOb, uOk := objMap["Url"]
		url, uStr := urlOb.(string)
		if !(uOk && uStr) {
			continue
		}
		tokenOb, tOk := objMap["Token"]
		token, tStr := tokenOb.(string)
		if !(tOk && tStr) {
			continue
		}
		if url == toUrl {
			SetAuth(GlobalOpts, toUrl, token)
			return true
		}
	}
	return false
}

func GetManualAuthMessage(GlobalOpts *CliOpts) string {
	// handle failure with manual copy/paste message (with URL)
	return fmt.Sprintf("ERROR: Automatic authentication token retrieval failed.\n") +
		fmt.Sprintf("Please retry or manually visit the authentication URL:\n") +
		fmt.Sprintf("\t"+GetTokenAuthUrl(GlobalOpts, true)+"\n")
}

func ValidateApiUrl(url string) bool {
	urlRegex := regexp.MustCompile(`^https?://[\.\:a-z0-9-]+$`)
	if !urlRegex.MatchString(url) {
		WriteMsg("ERROR: Invalid URL to auth! (%s)\n", url)
		return false
	}
	return true
}

// randomly generated key
func GetIdempotencyKey() string {
	// generate 128 high-quality random bytes, return as a hex string
	data := make([]byte, 16)
	hexBytes := make([]byte, 32)
	_, err := rand.Read(data[:])
	if err != nil {
		// Fall back to low-quality psuedo-random numbers.
		// NOTE: This is not ideal, but should be rare.
		if viper.GetBool("debug") {
			WriteMsg("WARNING: High-Quality random numbers unavailable. -- (%v)\n", err.Error())
		}
		prand.Seed(time.Now().UnixNano())
		for i := 0; i < 16; i++ {
			data[i] = byte(prand.Intn(256))
		}
	}
	hex.Encode(hexBytes, data)
	hexStr := string(hexBytes)
	return hexStr
}

func GetInnerErrorStr(errStr string) string {
	msgRegex := regexp.MustCompile(`message: \\".*\\"`)
	loc := msgRegex.FindStringIndex(errStr)
	if loc == nil {
		errStr = strings.Replace(errStr, "\\n", "\n", -1)
		return errStr
	}
	innerStr := errStr[loc[0]+11 : loc[1]-2]
	innerStr = strings.Replace(innerStr, "\\\"", "\"", -1)
	innerStr = strings.Replace(innerStr, "\\\\", "\\", -1)
	return GetInnerErrorStr(innerStr)
}

func GetInnerError(err error) string {
	outer := error.Error(err)
	innerStr := GetInnerErrorStr(outer)
	return innerStr
}

func ExecuteOpCommand(GlobalOpts *CliOpts, expr string) (string, error) {
	if !GlobalOpts.HasAuth {
		return "", fmt.Errorf("No valid auth credentials.")
	} else {
		if clientAuth == nil || clientAuth.BaseURL != GlobalOpts.Url || GlobalOpts.AuthChanged {
			// Auth data is persisted, so that we don't have to re-authorize for every command
			clientAuth = NewClientAuth(GlobalOpts.Url, GlobalOpts.Token, GetIdempotencyKey())
			GlobalOpts.AuthChanged = false
		} else {
			// Fresh Idempotency key for every command.
			clientAuth.ApiKey = GetIdempotencyKey()
		}
		fullExpr := expr
		new_client := NewClient(clientAuth)
		//fix this to be resolved input
		ret, error := new_client.Execute(fullExpr, false)
		if error != nil {
			inner := GetInnerError(error)
			return "", fmt.Errorf(inner)

		} else {
			retStr := string(ret)
			return retStr, nil
		}
	}
}

// Returns compressed base64 data, file size, md5 checksum.
func CompressedBase64(content []byte) string {
	// Create a writer that caches compressors.
	// For this operation type we supply a nil Reader.
	var encoder, _ = zstd.NewWriter(nil)

	// Compress a buffer.
	// If you have a destination buffer, the allocation in the call can also be eliminated.
	compressed := encoder.EncodeAll(content, make([]byte, 0, len(content)))

	encoded := base64.StdEncoding.EncodeToString(compressed)
	return encoded
}

func ContentMd5AndSize(content []byte) (string, int64) {
	// streaming md5sum
	hash := md5.New()
	hash.Write(content)
	md5Sum := fmt.Sprintf("%x", hash.Sum(nil))
	return md5Sum, int64(len(content))
}

func FileMd5AndSize(filename string) (error, string, int64) {
	fstat, err := os.Stat(filename)
	if err != nil || fstat.Size() == 0 { // skip non-existent or empty files
		return err, "", 0
	}
	fileLen := fstat.Size()
	file, err := os.Open(filename)
	if err != nil {
		return err, "", 0
	}
	defer file.Close()
	// streaming md5sum
	hash := md5.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return err, "", 0
	}
	md5Sum := fmt.Sprintf("%x", hash.Sum(nil))
	return nil, md5Sum, fileLen
}

// //////////////////////////////////////////////////////////
// //////////////////////////////////////////////////////////
func DownloadFileHttps(src string, dst string, token string) error {
	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("Couldn't open local download file '%s'\n", dst)
	}
	defer out.Close()

	resp, err := http.Get(src)
	if err != nil {
		return fmt.Errorf("Couldn't open download url '%s'\n", src)
	}
	defer resp.Body.Close()

	// NOTE: This processes a block at a time, which is important for large files and mem usage
	// TODO download to temp and mv/rm on success/fail
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("Couldn't process download data from url '%s'\n", src)
	}

	return nil
}

func UploadFileHttps(src string, dst string, token string) error {
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("couldn't open local upload file '%s'", src)
	}
	defer file.Close()

	stat, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("couldn't stat file to upload: %s", err.Error())
	}
	fileSize := stat.Size()

	reqOb, err := http.NewRequest(http.MethodPut, dst, file)
	if err != nil {
		fmt.Printf("couldn't create upload request object: %s", err.Error())
		return fmt.Errorf("couldn't create upload request object: %s", err.Error())
	}
	reqOb.Header.Set("x-ms-blob-type", "BlockBlob") // only used by Azure, ignored by S3
	reqOb.ContentLength = fileSize

	response, err := http.DefaultClient.Do(reqOb)
	if err != nil {
		fmt.Printf("couldn't upload file: %s", err.Error())
		return fmt.Errorf("couldn't upload file: %s", err.Error())
	}
	defer response.Body.Close()
	if response.StatusCode != 201 && response.StatusCode != 200 {
		var body []byte
		response.Body.Read(body)
		return fmt.Errorf("couldn't upload file, status: %s, message: %v", response.Status, string(body))
	}
	fmt.Printf("Uploaded file '%s' (%d bytes) status: %v - %v\n", src, fileSize, response.StatusCode, http.StatusText(response.StatusCode))

	return nil
}

func UploadFileHttpsFromString(data string, dst string, token string) error {
	f, err := os.CreateTemp("", "tmpfile-") // in Go version older than 1.17 you can use ioutil.TempFile
	if err != nil {
		return err
	}
	// close and remove the temporary file at the end of the program
	defer f.Close()
	defer os.Remove(f.Name())
	// write data to the temporary file
	if _, err := f.Write([]byte(data)); err != nil {
		return err
	}
	err = UploadFileHttps(f.Name(), dst, token)
	return err
}

func DeleteFileHttps(dst string, token string) error {
	resp, err := http.Get(dst)
	if err != nil {
		fmt.Printf("Couldn't open delete url '%s'\n", dst)
		return fmt.Errorf("Couldn't open delete url '%s'\n", dst)
	}
	defer resp.Body.Close()

	return nil
}

func DownloadFileHttpsToTemp(src string, token string) (string, error) {
	f, err := os.CreateTemp("", "tmp_shor_opcp-") // in Go version older than 1.17 you can use ioutil.TempFile
	if err != nil {
		return "", err
	}
	defer f.Close()
	// The caller is responsible for cleaning up the file.
	// XXX we could delete it and pass the handle to the caller...
	//defer os.Remove(f.Name())
	err = DownloadFileHttps(src, f.Name(), token)
	return f.Name(), err
}

////////////////////////////////////////////////////////////
////////////////////////////////////////////////////////////

//func ExtractPresignedUrl(jsonData []byte) (string, error) {
//	var objects interface{}
//	err := json.Unmarshal(jsonData, &objects)
//	if err != nil {
//		return "", fmt.Errorf("WARNING: Failed to parse JSON from ExtractPresignedUrl().")
//	}
//	url, ok := pretty.GetNestedValueOrDefault(objects, pretty.ToKeyPath("get_file_attribute"), nil).(string)
//	if !ok {
//		return "", fmt.Errorf("WARNING: Missing get_file_attribute key from ExtractPresignedUrl().\n")
//	}
//	return url, nil
//}
//
//func CheckOpCopyUriField(GlobalOpts *opts.CliOpts, symbolName string) (string, bool) {
//	timeout := viper.GetString("background_timeout")
//	pre_command := fmt.Sprintf("%s.uri", symbolName)
//	results_pre, err :=  ExecuteOpCommand(GlobalOpts, pre_command, timeout)
//	if viper.GetBool("debug") {
//		logging.WriteMsgColor(Red, "OpCp legacy test error: %v\n", err)
//	}
//	if err != nil {
//		return "", true
//	}
//	uri, err := ExtractPresignedUrl(results_pre)
//	if err != nil {
//		return "", true
//	}
//	// "get file attribute failed: field does not exist"
//	if strings.Contains(uri, "failed:") || strings.Contains(uri, "field does not exist") {
//		return "", true
//	}
//	// XXX could explicitly check for "s3:"/"gs:"/"https:"(AZ) or "/<symbolName>"
//	return uri, false
//}
//
//func PushOpCopyFileData(GlobalOpts *opts.CliOpts, symbolName string, fileName string) bool {
//	rpath, legacyPush := CheckOpCopyUriField(GlobalOpts, symbolName)
//	timeout := viper.GetString("background_timeout")
//	//pre_command := fmt.Sprintf("%s.uri", symbolName)
//	//results_pre, err :=  ExecuteOpCommand(GlobalOpts, pre_command, timeout)
//	//logging.WriteMsgColor(Red, "OpCp legacy test error: %v\n", err)
//	//if err != nil {
//	//	legacyPush = true
//	//}
//	base64Data := ""
//	if legacyPush {
//		ok := false
//		// deal with source filename being quoted (e.g. spaces in filename)
//		base64Data, ok, _, _ = FileToBase64(fileName, false)
//		if !ok {
//			logging.WriteMsgColor(Red, "OpCp failed failed get base64 data, symbol: %s\n", symbolName)
//			return false
//		}
//	} else {
//		command := fmt.Sprintf("%s.presigned_put", symbolName)
//		results, err :=  ExecuteOpCommand(GlobalOpts, command, timeout)
//		if err != nil {
//			logging.WriteMsgColor(Red, "OpCp failed failed call for presigned URL, symbol: %s\n", symbolName)
//			return false
//		}
//		//rpath, err := ExtractPresignedUrl(results_pre)
//		//if err != nil {
//		//	logging.WriteMsgColor(Red, "OpCp failed failed to extract remote path, symbol: %s\n", symbolName)
//		//	return false
//		//}
//		url, err := ExtractPresignedUrl(results)
//		if err != nil {
//			logging.WriteMsgColor(Red, "OpCp failed failed to extract presigned URL, symbol: %s\n", symbolName)
//			return false
//		}
//		// push to URL
//		logging.WriteMsgColor(Blue, "OpCp uploading to presigned URL, symbol: %s\n   %s\n", symbolName, url)
//		err = UploadFileHttps(fileName, url, "")
//		if err != nil {
//			logging.WriteMsgColor(Red, "OpCp failed failed to extract presigned URL, symbol: %s\n%s\n", symbolName, err.Error())
//			return false
//		}
//		base64Data = fmt.Sprintf(":%s", rpath)
//	}
//	// NOTE: even for S3 files, we need to set file_data to *something*
//	command := fmt.Sprintf("%s.file_data = \"%s\"", symbolName, base64Data)
//	if !HandleOpCommand(GlobalOpts, command, false) {
//		commandTrunc := command
//		if len(commandTrunc) > 200 {
//			commandTrunc = commandTrunc[0:200] + " ..."
//		}
//		logging.WriteMsgColor(Red, "OpCp failed at step: %s\n", commandTrunc)
//		return false
//	}
//	return true
//}
