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

func createBaseURL(mygengo MyGengo, method string) (theURL string) {
	var baseURL string
	if mygengo.Sandbox {
		baseURL = sandboxURL
	} else {
		baseURL = apiURL
	}
	theURL = baseURL + method
    return
}

func createGetURL(mygengo MyGengo, method string, authRequired bool,
	optionalParams map[string]string) (theURL string) {
	v := url.Values{}
	v.Set("api_key", mygengo.PublicKey)
	if authRequired {
		apiSig, currentTime := apiSigAndCurrentTs(mygengo)
		v.Set("api_sig", apiSig)
		v.Set("ts", currentTime)
	}
	for key, val := range optionalParams {
		v.Set(key, val)
	}
    baseURL := createBaseURL(mygengo, method)
	s := []string{baseURL, "?", v.Encode()}
	theURL = strings.Join(s, "")
	return
}

func getRequest(method string, mygengo MyGengo, authRequired bool,
    optionalParams map[string]string) (theJSON interface{}) {
	theURL := createGetURL(mygengo, method, authRequired, optionalParams)
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

func postOrPutRequest(postOrPut string, method string, mygengo MyGengo, data string) (theJSON interface{}) {
    theURL := createBaseURL(mygengo, method)
	apiSig, currentTime := apiSigAndCurrentTs(mygengo)

    v := url.Values{}
	v.Set("api_key", mygengo.PublicKey)
	v.Set("api_sig", apiSig)
	v.Set("ts", currentTime)
    v.Set("data", data)

    client := &http.Client{}
    req, err := http.NewRequest(postOrPut, theURL, strings.NewReader(v.Encode()))
    req.Header.Add("Accept", "application/json")
    req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
    resp, err := client.Do(req)
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

func postRequest(method string, mygengo MyGengo, data string) (theJSON interface{}) {
    return postOrPutRequest("POST", method, mygengo, data)
}

func putRequest(method string, mygengo MyGengo, data string) (theJSON interface{}) {
    return postOrPutRequest("PUT", method, mygengo, data)
}

func (mygengo *MyGengo) AccountStats() interface{} {
    return getRequest("account/stats", *mygengo, true, nil)
}

func (mygengo *MyGengo) AccountBalance() interface{} {
    return getRequest("account/balance", *mygengo, true, nil)
}

func (mygengo *MyGengo) JobRevision(jobId int, revisionId int) interface{} {
    method := fmt.Sprintf("translate/job/%d/revision/%d", jobId, revisionId)
    return getRequest(method, *mygengo, true, nil)
}

func (mygengo *MyGengo) JobRevisions(jobId int) interface{} {
    method := fmt.Sprintf("translate/job/%d/revisions", jobId)
    return getRequest(method, *mygengo, true, nil)
}

func (mygengo *MyGengo) JobFeedback(jobId int) interface{} {
    method := fmt.Sprintf("translate/job/%d/feedback", jobId)
    return getRequest(method, *mygengo, true, nil)
}

func (mygengo *MyGengo) JobComments(jobId int) interface{} {
    method := fmt.Sprintf("translate/job/%d/comments", jobId)
    return getRequest(method, *mygengo, true, nil)
}

func (mygengo *MyGengo) Job(jobId int, optionalParams map[string]string) interface{} {
    method := fmt.Sprintf("translate/job/%d", jobId)
    return getRequest(method, *mygengo, true, optionalParams)
}

func (mygengo *MyGengo) JobsGroup(groupId int) interface{} {
    method := fmt.Sprintf("translate/jobs/group/%d", groupId)
    return getRequest(method, *mygengo, true, nil)
}

func (mygengo *MyGengo) Jobs(optionalParams map[string]string) interface{} {
    method := "translate/jobs"
    return getRequest(method, *mygengo, true, optionalParams)
}

func (mygengo *MyGengo) JobsByIds(jobIds []int) interface{} {
    jobIdsStrings := []string{}
    for _, jobId := range jobIds {
        jobIdsStrings = append(jobIdsStrings, strconv.Itoa(jobId))
    }
    jobIdsString := strings.Join(jobIdsStrings, ",")
    method := fmt.Sprintf("translate/jobs/%s", jobIdsString)
    return getRequest(method, *mygengo, true, nil)
}

func (mygengo *MyGengo) Languages() interface{} {
    method := "translate/service/languages"
    return getRequest(method, *mygengo, false, nil)
}

func (mygengo *MyGengo) LanguagePairs(optionalParams map[string]string) interface{} {
    method := "translate/service/language_pairs"
    return getRequest(method, *mygengo, false, optionalParams)
}

func (mygengo *MyGengo) PostJobComment(jobId int, comment string) interface{} {
    method := fmt.Sprintf("translate/job/%d/comment", jobId)
    type Comment struct {
        Body string `json:"body"`
    }
    commentJSON, err := json.Marshal(Comment{Body: comment})
    if err != nil {
        fmt.Println(err)
    }
    return postRequest(method, *mygengo, string(commentJSON))
}

type ReviseAction struct {
    Action string `json:"action"`
    Comment string `json:"comment"`
}

func NewReviseAction(comment string) (reviseAction ReviseAction) {
    reviseAction = ReviseAction{Action: "revise",
                                Comment: comment}
    return
}

func (mygengo *MyGengo) ReviseJob(jobId int, reviseAction ReviseAction) interface{} {
    method := fmt.Sprintf("translate/job/%d", jobId)
    reviseActionJSON, err := json.Marshal(reviseAction)
    if err != nil {
        fmt.Println(err)
    }
    return putRequest(method, *mygengo, string(reviseActionJSON))
}

type ApproveAction struct {
    Action string `json:"action"`
    Rating *int `json:"rating,omitempty"`
    ForTranslator *string `json:"for_translator,omitempty"`
    ForMyGengo *string `json:"for_mygengo,omitempty"`
    Public *int `json:"public,omitempty"`
}

func NewApproveAction() (approveAction ApproveAction) {
    approveAction = ApproveAction{Action: "approve"}
    return
}

func (a *ApproveAction) addRating(rating int) {
    a.Rating = &rating
}

func (a *ApproveAction) addForTranslator(forTranslator string) {
    a.ForTranslator = &forTranslator
}

func (a *ApproveAction) addForMyGengo(forMyGengo string) {
    a.ForMyGengo = &forMyGengo
}

func (a *ApproveAction) addPublic(public int) {
    a.Public = &public
}

func (mygengo *MyGengo) ApproveJob(jobId int, approveAction ApproveAction) interface{} {
    method := fmt.Sprintf("translate/job/%d", jobId)
    approveActionJSON, err := json.Marshal(approveAction)
    if err != nil {
        fmt.Println(err)
    }
    return putRequest(method, *mygengo, string(approveActionJSON))
}

func main() {
}
