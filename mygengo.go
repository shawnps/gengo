package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/url"
	"strconv"
	"time"
	//        "net/http"
	"strings"

//        "encoding/json"
)

const (
	sandboxURL = "http://api.sandbox.mygengo.com/v1.1/"
	apiURL     = "http://api.mygengo.com/v1.1/"
)

type MyGengo struct {
	PublicKey  string
	PrivateKey string
	Sandbox    bool
}

func hmacSha1Hex(key string, aString string) string {
	hasher := hmac.New(sha1.New, []byte(key))
	hasher.Write([]byte(aString))
	return hex.EncodeToString(hasher.Sum(nil))
}

func apiSigAndCurrentTs(myGengo MyGengo) (apiSig string, timestamp string) {
	now := time.Now()
	currentTs := now.Unix()
	timestamp = strconv.FormatInt(currentTs, 10)
	apiSig = hmacSha1Hex(myGengo.PrivateKey, timestamp)
	return
}

func createURL(mygengo MyGengo, method string, authRequired bool,
	optionalParams map[string]string) (theURL string) {
	v := url.Values{}
	var baseURL string
	if mygengo.Sandbox {
		baseURL = sandboxURL
	} else {
		baseURL = apiURL
	}
	v.Set("api_key", mygengo.PublicKey)
	if authRequired {
		apiSig, currentTime := apiSigAndCurrentTs(mygengo)
		v.Set("api_sig", apiSig)
		v.Set("ts", currentTime)
	}
	for key, val := range optionalParams {
		v.Set(key, val)
	}
	s := []string{baseURL, method, "?", v.Encode()}
	theURL = strings.Join(s, "")
	return theURL
}
