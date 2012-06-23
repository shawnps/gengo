package mygengo

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
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

func createGetOrDeleteURL(mygengo MyGengo, method string, authRequired bool,
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

func readAndUnmarshalResponse(resp http.Response) (theJSON interface{}) {
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(body, &theJSON)
	if err != nil {
		log.Fatal(err)
	}
	return theJSON
}

func doGetOrDelete(getOrDelete, url string) interface{} {
	client := &http.Client{}
	req, err := http.NewRequest(getOrDelete, url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	return readAndUnmarshalResponse(*resp)
}

func getRequest(method string, mygengo MyGengo, authRequired bool,
	optionalParams map[string]string) interface{} {
	theURL := createGetOrDeleteURL(mygengo, method, authRequired, optionalParams)
	return doGetOrDelete("GET", theURL)
}

func getRequestForImage(method string, mygengo MyGengo, fileName string) (err error) {
	theURL := createGetOrDeleteURL(mygengo, method, true, nil)
	resp, err := http.Get(theURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	dst, err := os.Create(fileName)
	if err != nil {
		return
	}
	defer dst.Close()
	io.Copy(dst, resp.Body)
	if err != nil {
		return
	}
	return nil
}

func postOrPutRequest(postOrPut string, method string, mygengo MyGengo, data string) interface{} {
	theURL := createBaseURL(mygengo, method)
	apiSig, currentTime := apiSigAndCurrentTs(mygengo)

	v := url.Values{}
	v.Set("api_key", mygengo.PublicKey)
	v.Set("api_sig", apiSig)
	v.Set("ts", currentTime)
	v.Set("data", data)

	client := &http.Client{}
	req, err := http.NewRequest(postOrPut, theURL, strings.NewReader(v.Encode()))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	return readAndUnmarshalResponse(*resp)
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

func (mygengo *MyGengo) JobPreview(jobId int, fileName string) error {
	method := fmt.Sprintf("translate/job/%d/preview", jobId)
	return getRequestForImage(method, *mygengo, fileName)
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

func (mygengo *MyGengo) PostJobComment(jobId int, comment string) interface{} {
	method := fmt.Sprintf("translate/job/%d/comment", jobId)
	var postComment struct {
		Body string `json:"body"`
	}
	commentJSON, err := json.Marshal(postComment{Body: comment})
	if err != nil {
		log.Fatal(err)
	}
	return postRequest(method, *mygengo, string(commentJSON))
}

func (mygengo *MyGengo) JobComments(jobId int) interface{} {
	method := fmt.Sprintf("translate/job/%d/comments", jobId)
	return getRequest(method, *mygengo, true, nil)
}

func (mygengo *MyGengo) DeleteJob(jobId int) interface{} {
	method := fmt.Sprintf("translate/job/%d", jobId)
	theURL := createGetOrDeleteURL(*mygengo, method, true, nil)
	return doGetOrDelete("DELETE", theURL)
}

func (mygengo *MyGengo) Job(jobId int, optionalParams map[string]string) interface{} {
	method := fmt.Sprintf("translate/job/%d", jobId)
	return getRequest(method, *mygengo, true, optionalParams)
}

type ReviseAction struct {
	ActionType string `json:"action"`
	Comment    string `json:"comment"`
}

func NewReviseAction(comment string) (reviseAction ReviseAction) {
	reviseAction = ReviseAction{ActionType: "revise",
		Comment: comment}
	return
}

func (mygengo *MyGengo) ReviseJob(jobId int, reviseAction ReviseAction) interface{} {
	method := fmt.Sprintf("translate/job/%d", jobId)
	reviseActionJSON, err := json.Marshal(reviseAction)
	if err != nil {
		log.Fatal(err)
	}
	return putRequest(method, *mygengo, string(reviseActionJSON))
}

type ApproveAction struct {
	ActionType    string  `json:"action"`
	Rating        *int    `json:"rating,omitempty"`
	ForTranslator *string `json:"for_translator,omitempty"`
	ForMyGengo    *string `json:"for_mygengo,omitempty"`
	Public        *int    `json:"public,omitempty"`
}

func NewApproveAction() (approveAction ApproveAction) {
	approveAction = ApproveAction{ActionType: "approve"}
	return
}

func (approveAction *ApproveAction) addRating(rating int) {
	approveAction.Rating = &rating
}

func (approveAction *ApproveAction) addForTranslator(forTranslator string) {
	approveAction.ForTranslator = &forTranslator
}

func (approveAction *ApproveAction) addForMyGengo(forMyGengo string) {
	approveAction.ForMyGengo = &forMyGengo
}

func (approveAction *ApproveAction) addPublic(public int) {
	approveAction.Public = &public
}

func (mygengo *MyGengo) ApproveJob(jobId int, approveAction ApproveAction) interface{} {
	method := fmt.Sprintf("translate/job/%d", jobId)
	approveActionJSON, err := json.Marshal(approveAction)
	if err != nil {
		log.Fatal(err)
	}
	return putRequest(method, *mygengo, string(approveActionJSON))
}

type RejectAction struct {
	ActionType string  `json:"action"`
	Reason     string  `json:"reason"`
	Comment    string  `json:"comment"`
	Captcha    string  `json:"captcha"`
	FollowUp   *string `json:"follow_up,omitempty"`
}

func NewRejectAction(reason string, comment string, captcha string) (rejectAction RejectAction) {
	rejectAction = RejectAction{ActionType: "reject",
		Reason:  reason,
		Comment: comment,
		Captcha: captcha}
	return
}

func (rejectAction *RejectAction) addFollowUp(followUp string) {
	rejectAction.FollowUp = &followUp
}

func (mygengo *MyGengo) RejectJob(jobId int, rejectAction RejectAction) interface{} {
	method := fmt.Sprintf("translate/job/%d", jobId)
	rejectActionJSON, err := json.Marshal(rejectAction)
	if err != nil {
		log.Fatal(err)
	}
	return putRequest(method, *mygengo, string(rejectActionJSON))
}

type JobPayload struct {
	BodySrc      string  `json:"body_src"`
	LcSrc        string  `json:"lc_src"`
	LcTgt        string  `json:"lc_tgt"`
	Tier         string  `json:"tier"`
	Force        *int    `json:"force,omitempty"`
	Comment      *string `json:"comment,omitempty"`
	UsePreferred *int    `json:"use_preferred,omitempty"`
	CallbackURL  *string `json:"callback_url,omitempty"`
	AutoApprove  *int    `json:"auto_approve,omitempty"`
	CustomData   *string `json:"custom_data,omitempty"`
}

func NewJobPayload(bodySrc string, lcSrc string, lcTgt string, tier string) (jobPayload JobPayload) {
	jobPayload = JobPayload{BodySrc: bodySrc,
		LcSrc: lcSrc,
		LcTgt: lcTgt,
		Tier:  tier}
	return
}

func (jobPayload *JobPayload) addForce(force int) {
	jobPayload.Force = &force
}

func (jobPayload *JobPayload) addComment(comment string) {
	jobPayload.Comment = &comment
}

func (jobPayload *JobPayload) addUsePreferred(usePreferred int) {
	jobPayload.UsePreferred = &usePreferred
}

func (jobPayload *JobPayload) addCallbackURL(callbackURL string) {
	jobPayload.CallbackURL = &callbackURL
}

func (jobPayload *JobPayload) addAutoApprove(autoApprove int) {
	jobPayload.AutoApprove = &autoApprove
}

func (jobPayload *JobPayload) addCustomData(customData string) {
	jobPayload.CustomData = &customData
}

func (mygengo *MyGengo) PostJob(jobPayload JobPayload) interface{} {
	type Job struct {
		JobPayload JobPayload `json:"job"`
	}
	method := "translate/job"
	job := Job{JobPayload: jobPayload}
	postJobJSON, err := json.Marshal(job)
	fmt.Println(string(postJobJSON))
	if err != nil {
		log.Fatal(err)
	}
	return postRequest(method, *mygengo, string(postJobJSON))
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

type JobArray struct {
	Jobs    []JobPayload `json:"jobs"`
	AsGroup *int         `json:"as_group,omitempty"`
}

func (jobArray *JobArray) addAsGroup(asGroup int) {
	jobArray.AsGroup = &asGroup
}

func NewJobArray(jobs []JobPayload) (jobArray JobArray) {
	jobArray = JobArray{Jobs: jobs}
	return
}

func (mygengo *MyGengo) PostJobs(jobArray JobArray) interface{} {
	method := "translate/jobs"
	postJobsJSON, err := json.Marshal(jobArray)
	if err != nil {
		log.Fatal(err)
	}
	return postRequest(method, *mygengo, string(postJobsJSON))
}

func (mygengo *MyGengo) LanguagePairs(optionalParams map[string]string) interface{} {
	method := "translate/service/language_pairs"
	return getRequest(method, *mygengo, false, optionalParams)
}

func (mygengo *MyGengo) Languages() interface{} {
	method := "translate/service/languages"
	return getRequest(method, *mygengo, false, nil)
}

func (mygengo *MyGengo) JobsQuote(jobArray JobArray) interface{} {
	method := "translate/service/quote"
	jobsQuoteJSON, err := json.Marshal(jobArray)
	if err != nil {
		log.Fatal(err)
	}
	return postRequest(method, *mygengo, string(jobsQuoteJSON))
}
