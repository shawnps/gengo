package main

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
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
	return
}

func getRequest(method string, mygengo MyGengo, authRequired bool,
    optionalParams map[string]string) (theJSON interface{}) {
	theURL := createURL(mygengo, method, authRequired, optionalParams)
	client := &http.Client{}
	req, err := http.NewRequest("GET", theURL, nil)
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)
    if err != nil {
        fmt.Println(err)
        return
    }
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        fmt.Println(err)
        return
    }
    err = json.Unmarshal(body, &theJSON)
    if err != nil {
        fmt.Println(err)
        return
    }
    return
}

func (mygengo *MyGengo) getAccountStats() interface{} {
    return getRequest("account/stats", *mygengo, true, nil)
}

func (mygengo *MyGengo) getAccountBalance() interface{} {
    return getRequest("account/balance", *mygengo, true, nil)
}

func (mygengo *MyGengo) getJobRevision(jobId int, revisionId int) interface{} {
    method := fmt.Sprintf("translate/job/%d/revision/%d", jobId, revisionId)
    return getRequest(method, *mygengo, true, nil)
}

func main() {
}
