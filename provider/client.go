// Copyright 2025 NVIDIA Corporation
// SPDX-License-Identifier: Apache-2.0

package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// The endpoint path to execute command.
const executeEndpoint = "/api/v1/execute"
const authEndpoint = "/api/v1/token/refresh?flow_type=cli"
const requestTimeoutSec = 90
const accessTokenTTL = 60 * 60 // one hour expiration for CLI access tokens

func GetTokenAuthUrl(GlobalOpts *CliOpts, manual bool) string {
	// NOTE: there should be no trailing "/" on GlobalOpts.Url
	if manual {
		return GlobalOpts.Url + "/sso/v2/saml/auth_redirect?type=saml&mode=copy_paste"
	} else {
		return GlobalOpts.Url + "/sso/v2/saml/auth_redirect?type=saml&mode=download"
	}
}

type AuthTokenStruct struct {
	Raw       string
	Parts     []string
	AlgoStr   string
	ClaimStr  string
	Claim     map[string]interface{}
	Customer  string
	User      string
	Expiry    int64
	ExpiryStr string
	Type      string
}

func AuthStructToMap(ats AuthTokenStruct) map[string]string {
	theMap := map[string]string{}
	theMap["raw"] = ats.Raw
	theMap["algo"] = ats.AlgoStr
	theMap["claim"] = ats.ClaimStr
	theMap["customer"] = ats.Customer
	theMap["user"] = ats.User
	theMap["expiry"] = ats.ExpiryStr
	theMap["type"] = ats.Type
	return theMap
}

func DecodeAuthToken(token string) *AuthTokenStruct {
	ats := AuthTokenStruct{}
	ats.Parts = strings.Split(token, ".")
	if len(ats.Parts) < 3 {
		WriteMsg("Incomplete Auth Token! (%s)\n", token)
		return nil
	}
	//ats.ClaimStr, err = base64.URLEncoding.DecodeString(ats.Parts[1])
	algoStr, err := base64.RawURLEncoding.DecodeString(ats.Parts[0])
	if err != nil {
		WriteMsg("Failed to decode Auth Token Algo! %v (%v)\n", err, ats.Parts[0])
		return nil
	}
	claimStr, err := base64.RawURLEncoding.DecodeString(ats.Parts[1])
	if err != nil {
		WriteMsg("Failed to decode Auth Token Claim! %v (%v)\n", err, ats.Parts[1])
		return nil
	}
	_, err = base64.RawURLEncoding.DecodeString(ats.Parts[2])
	if err != nil {
		WriteMsg("Failed to decode Auth Token Signature! %v (%v)\n", err, ats.Parts[2])
		return nil
	}
	ats.AlgoStr = string(algoStr)
	ats.ClaimStr = string(claimStr)
	//WriteMsg("Decoded auth -> %v\n", string(ats.ClaimStr))
	ats.Claim = map[string]interface{}{}
	err = json.Unmarshal([]byte(ats.ClaimStr), &ats.Claim)
	if err != nil {
		WriteMsg("Auth Token claim is invalid JSON! %v (%v)\n", err, ats.ClaimStr)
		return nil
	}
	ats.Customer = CastToString(GetNestedValueOrDefault(ats.Claim, ToKeyPath("cst"), ""))
	ats.User = CastToString(GetNestedValueOrDefault(ats.Claim, ToKeyPath("sub"), ""))
	ats.Type = CastToString(GetNestedValueOrDefault(ats.Claim, ToKeyPath("aud"), ""))
	expiry := GetNestedValueOrDefault(ats.Claim, ToKeyPath("exp"), 0)
	ats.Expiry = int64(expiry.(float64))
	t := time.Unix(ats.Expiry, 0)
	ats.ExpiryStr = t.Format(time.UnixDate)
	return &ats
}

type ClientAuth struct {
	BaseURL      string
	ApiToken     string
	AccessToken  string
	AccessExpiry int64
	ApiKey       string
}

// Client client for sending request to opslang backend service
type Client struct {
	httpClient *http.Client
	authData   *ClientAuth
}

type clientOption func(*Client)

// NOTE: you can run with "GODEBUG=http2debug=2" on the commandline to print out client debugging info.

// NewClient OpslangClient instance
func NewClientAuth(host string, apiToken string, apiKey string, options ...clientOption) *ClientAuth {
	auth := &ClientAuth{
		BaseURL:  host,
		ApiToken: apiToken,
		ApiKey:   apiKey,
	}
	return auth
}

// NewClient OpslangClient instance
func NewClient(auth *ClientAuth, options ...clientOption) *Client {
	client := &Client{
		httpClient: &http.Client{
			Timeout: time.Second * requestTimeoutSec,
		},
		authData: auth,
	}

	for i := range options {
		options[i](client)
	}

	return client
}

func setHTTPClientOption(httpClient *http.Client) clientOption {
	return func(client *Client) {
		client.httpClient = httpClient
	}
}

// Execute sends statement to platform backend
func (client *Client) Execute(statement string, suppressErrors bool) (ret []byte, err error) {
	if !client.maybeRefreshAccessToken(suppressErrors) {
		return []byte(""), fmt.Errorf("Access token refresh failed.")
	}
	ret, err, code := client.executeInner(statement, suppressErrors)
	if code == 401 {
		// Second chance (in case latency/etc causes an expired token).
		// Force a token refresh
		client.authData.AccessExpiry = 0
		if !client.maybeRefreshAccessToken(suppressErrors) {
			return []byte(""), err
		}
		ret, err, code = client.executeInner(statement, suppressErrors)
	}
	return ret, err
}

func (client *Client) maybeRefreshAccessToken(suppressErrors bool) bool {
	decoded := DecodeAuthToken(client.authData.ApiToken)
	if decoded == nil {
		if viper.GetBool("debug") {
			WriteMsg("ApiToken is invalid.\n")
		}
		return false
	}
	if decoded.Type == "access" {
		now := time.Now().Unix()
		if viper.GetBool("debug") {
			WriteMsg("ApiToken is an access token (not refresh) using it directly.\n")
		}
		if decoded.Expiry <= now {
			if !suppressErrors {
				WriteMsg("ApiToken is an access token (not refresh) but has expired.\n")
			}
			return false
		}
		client.authData.AccessToken = client.authData.ApiToken
		return true
	}
	now := time.Now().Unix()
	// To avoid the latency of getting an access token on every op-statement:
	//   keep a timestamp (1 hour expiry)
	//   only refresh if past expiry, or executeInner() returns 401
	if client.authData.AccessExpiry <= now || client.authData.AccessToken == "" {
		if viper.GetBool("debug") {
			WriteMsg("Re-Authorizing... (%d - %d = %d) token: '%s'\n", client.authData.AccessExpiry, now, now-client.authData.AccessExpiry, client.authData.AccessToken)
		}
		auth, err := client.fetchAccessToken(suppressErrors)
		if err != nil {
			return false
		}
		client.authData.AccessToken = string(auth)
		// intentionally use old "now" to account for network delays
		client.authData.AccessExpiry = now + accessTokenTTL
	}
	return true
}

func maybePrintTimer(startTimeMs int64, label string) {
	if viper.GetBool("debug") || viper.GetBool("timing") {
		endTimeMs := time.Now().UnixNano() / 1_000_000
		WriteMsg("-- Executed %s in %d ms --\n", label, endTimeMs-startTimeMs)
	}
}

func (client *Client) callApi(suppressErrors bool, auth string, url string, body string, kind string) (ret []byte, err error, code int) {
	startTimeMs := time.Now().UnixNano() / 1_000_000
	defer maybePrintTimer(startTimeMs, kind)

	authorization := fmt.Sprintf("Bearer %s", auth)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(body)))
	if err != nil {
		if !suppressErrors {
			WriteMsg("ERROR creating HTTP request object.\n")
		}
		return ret, err, 0
	}
	req.Header.Set("authorization", authorization)
	req.Header.Set("content-type", "application/json; charset=utf-8")
	req.Header.Set("idempotency-key", client.authData.ApiKey)
	req.Header.Set("accept", "*/*")

	// Allow Ctrl-C to cancel
	canceled := false
	ctx, cancel := context.WithCancel(context.Background())
	req = req.WithContext(ctx)
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer func() {
		signal.Stop(interrupt) //, os.Interrupt)
		close(interrupt)
	}()
	go func() {
		sig := <-interrupt
		canceled = true
		if sig != nil {
			//fmt.Printf("cancel signal %v\n", sig)
			cancel()
		}
	}()

	resp, err := client.httpClient.Do(req)
	if canceled {
		return ret, errors.New("User Cancelled"), 0
	}

	if err != nil {
		if !suppressErrors {
			WriteMsg("ERROR fetching HTTP response -- %s.\n", kind)
		}
		return ret, err, 0
	}

	defer func() {
		if resp.Body != nil {
			if err := resp.Body.Close(); err != nil {
				if !suppressErrors {
					WriteMsg("ERROR: Closing HTTP connection: %s\n", err)
				}
			}
		}
	}()

	if resp.Body == nil {
		err = fmt.Errorf("ERROR: Empty HTTP response body -- %s.\n", kind)
		if !suppressErrors {
			WriteMsg("%s", err.Error())
		}
		return ret, err, 0
	}

	ret, err = ioutil.ReadAll(resp.Body)

	if err != nil {
		if !suppressErrors {
			WriteMsg("ERROR Reading HTTP response -- %s.\n", kind)
		}
		return ret, err, 0
	}

	return ret, err, resp.StatusCode
}

func (client *Client) fetchAccessToken(suppressErrors bool) (ret []byte, err error) {
	url := fmt.Sprintf("%s%s", client.authData.BaseURL, authEndpoint)
	auth := client.authData.ApiToken
	kind := "fetchAccessToken()"
	body := "{\"refresh_token\": \"" + client.authData.ApiToken + "\"}"
	ret, err, code := client.callApi(suppressErrors, auth, url, body, kind)

	if code != 200 {
		if !suppressErrors {
			WriteMsg("ERROR Unexpected HTTP status code (%v) in auth response.\n", code)
			WriteMsg("You may need to get a fresh authorization token! e.g\n")
			WriteMsg(" 'auth %s'\n", client.authData.BaseURL)
		}
		return ret, fmt.Errorf(string(ret))
	}

	var js interface{}
	jsErr := json.Unmarshal(ret, &js)
	if jsErr != nil {
		if !suppressErrors {
			WriteMsg("ERROR Unmarshaling HTTP auth response.\n")
		}
		return ret, err
	}
	// NOTE: A refresh token is also returned, so we could also update the token saved in the config.
	//refresh, isStr := GetNestedValueOrDefault(js, ToKeyPath("refresh_token"), "").(string)
	access, isStr := GetNestedValueOrDefault(js, ToKeyPath("access_token"), "").(string)
	if !isStr || access == "" {
		if !suppressErrors {
			WriteMsg("ERROR Missing token in auth response.\n")
		}
		return ret, fmt.Errorf("Missing access token in response.")
	}

	return []byte(access), err
}

func (client *Client) executeInner(statement string, suppressErrors bool) (ret []byte, err error, code int) {
	url := fmt.Sprintf("%s%s", client.authData.BaseURL, executeEndpoint)
	auth := client.authData.AccessToken
	kind := "Execute()"
	body_data := map[string]string{"statement": statement}
	body, err := json.Marshal(body_data)
	if err != nil {
		if !suppressErrors {
			WriteMsg("ERROR marshaling op statement body.\n")
		}
		return ret, err, 0
	}
	ret, err, code = client.callApi(suppressErrors, auth, url, string(body), kind)

	if code != 200 {
		if ret == nil || len(ret) == 0 {
			ret = []byte(fmt.Sprintf("ERROR: Unexpected HTTP status code (%v) in response.\n", code))
		}
		return ret, fmt.Errorf("%s", string(ret)), code
	}

	return ret, err, code
}
