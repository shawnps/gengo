package gengo

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	sandboxURL = "http://api.sandbox.mygengo.com/v2/"
	apiURL     = "http://api.gengo.com/v2/"
)

type Gengo struct {
	PublicKey  string
	PrivateKey string
	Sandbox    bool
}

func hmacSha1Hex(key string, aString string) string {
	hasher := hmac.New(sha1.New, []byte(key))
	hasher.Write([]byte(aString))
	return hex.EncodeToString(hasher.Sum(nil))
}

func apiSigAndCurrentTs(gengo Gengo) (apiSig string, timestamp string) {
	now := time.Now()
	currentTs := now.Unix()
	timestamp = strconv.FormatInt(currentTs, 10)
	apiSig = hmacSha1Hex(gengo.PrivateKey, timestamp)
	return
}

func createBaseURL(gengo Gengo, method string) (theURL string) {
	var baseURL string
	if gengo.Sandbox {
		baseURL = sandboxURL
	} else {
		baseURL = apiURL
	}
	theURL = baseURL + method
	return
}

func createGetOrDeleteURL(gengo Gengo, method string, authRequired bool,
	optionalParams map[string]string) (theURL string) {
	v := url.Values{}
	v.Set("api_key", gengo.PublicKey)
	if authRequired {
		apiSig, currentTime := apiSigAndCurrentTs(gengo)
		v.Set("api_sig", apiSig)
		v.Set("ts", currentTime)
	}
	for key, val := range optionalParams {
		v.Set(key, val)
	}
	baseURL := createBaseURL(gengo, method)
	s := []string{baseURL, "?", v.Encode()}
	theURL = strings.Join(s, "")
	return
}

func doGetOrDelete(getOrDelete, url string) (body []byte) {
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
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return
}

func getRequest(method string, gengo Gengo, authRequired bool,
	optionalParams map[string]string) []byte {
	theURL := createGetOrDeleteURL(gengo, method, authRequired, optionalParams)
	return doGetOrDelete("GET", theURL)
}

func postOrPutRequest(postOrPut string, method string, gengo Gengo, data string) (body []byte) {
	theURL := createBaseURL(gengo, method)
	apiSig, currentTime := apiSigAndCurrentTs(gengo)

	v := url.Values{}
	v.Set("api_key", gengo.PublicKey)
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
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	return
}

func postRequest(method string, gengo Gengo, data string) []byte {
	return postOrPutRequest("POST", method, gengo, data)
}

func putRequest(method string, gengo Gengo, data string) []byte {
	return postOrPutRequest("PUT", method, gengo, data)
}

// For when opstat is "error"
type FailedResponse struct {
	Code int
	Msg  string
}

// The API returns strings for some things that one would expect
// to be numbers (for example, credits_spent)
// FloatString takes a string and converts it to a float64.
type FloatString string

func (f *FloatString) UnmarshalJSON(i interface{}) (n float64) {
	s := i.(string)
	n, err := strconv.ParseFloat(s, 64)
	if err != nil {
		log.Fatal(err)
	}
	return
}

// Same as FloatString, but for integers.
type IntString string

func (f *IntString) UnmarshalJSON(i interface{}) (n int64) {
	s := i.(string)
	n, err := strconv.ParseInt(s, 10, 0)
	if err != nil {
		log.Fatal(err)
	}
	return
}

type AccountStatsResponse struct {
	Opstat   string
	Response struct {
		UserSince    int64       `json:"user_since"`
		CreditsSpent FloatString `json:"credits_spent"`
		Currency     string
	}
	Err *FailedResponse
}

func (gengo *Gengo) AccountStats() (r *AccountStatsResponse, err error) {
	b := getRequest("account/stats", *gengo, true, nil)
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return nil, err
	}
	return
}

type AccountBalanceResponse struct {
	Opstat   string
	Response struct {
		Credits  FloatString
		Currency *string
	}
	Err *FailedResponse
}

func (gengo *Gengo) AccountBalance() (r *AccountBalanceResponse, err error) {
	b := getRequest("account/balance", *gengo, true, nil)
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return nil, err
	}
	return
}

type JobRevisionResponse struct {
	Opstat   string
	Response struct {
		Revision struct {
			Ctime   int64
			BodyTgt *string `json:"body_tgt"`
		}
	}
	Err *FailedResponse
}

func (gengo *Gengo) JobRevision(jobId int, revisionId int) (r *JobRevisionResponse, err error) {
	method := fmt.Sprintf("translate/job/%d/revision/%d", jobId, revisionId)
	b := getRequest(method, *gengo, true, nil)
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return nil, err
	}
	return
}

type JobRevisionsResponse struct {
	Opstat   string
	Response struct {
		JobId     IntString `json:"job_id"`
		Revisions []struct {
			Ctime int64
			RevId IntString `json:"rev_id"`
		}
	}
	Err *FailedResponse
}

func (gengo *Gengo) JobRevisions(jobId int) (r *JobRevisionsResponse, err error) {
	method := fmt.Sprintf("translate/job/%d/revisions", jobId)
	b := getRequest(method, *gengo, true, nil)
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return nil, err
	}
	return
}

type JobFeedbackResponse struct {
	Opstat   string
	Response struct {
		Feedback struct {
			Rating        FloatString
			ForTranslator *string `json:"for_translator"`
		}
	}
	Err *FailedResponse
}

func (gengo *Gengo) JobFeedback(jobId int) (r *JobFeedbackResponse, err error) {
	method := fmt.Sprintf("translate/job/%d/feedback", jobId)
	b := getRequest(method, *gengo, true, nil)
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return nil, err
	}
	return
}

type EmptyResponse struct {
	Opstat string
	Err    *FailedResponse
}

func (gengo *Gengo) PostJobComment(jobId int, comment string) (err error) {
	method := fmt.Sprintf("translate/job/%d/comment", jobId)
	var postComment struct {
		Body string `json:"body"`
	}
	postComment.Body = comment
	commentJSON, err := json.Marshal(postComment)
	if err != nil {
		return err
	}
	b := postRequest(method, *gengo, string(commentJSON))
	var r EmptyResponse
	err = json.Unmarshal(b, &r)
	if err != nil {
		return err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return err
	}
	return
}

type JobCommentsResponse struct {
	Opstat   string
	Response struct {
		Thread []struct {
			Author string
			Body   string
			Ctime  int64
		}
	}
	Err *FailedResponse
}

func (gengo *Gengo) JobComments(jobId int) (r *JobCommentsResponse, err error) {
	method := fmt.Sprintf("translate/job/%d/comments", jobId)
	b := getRequest(method, *gengo, true, nil)
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return nil, err
	}
	return
}

func (gengo *Gengo) DeleteJob(jobId int) (err error) {
	method := fmt.Sprintf("translate/job/%d", jobId)
	theURL := createGetOrDeleteURL(*gengo, method, true, nil)
	b := doGetOrDelete("DELETE", theURL)
	var r EmptyResponse
	err = json.Unmarshal(b, &r)
	if err != nil {
		return err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return err
	}
	return
}

type JobResponse struct {
	Opstat   string
	Response struct {
		Job struct {
			AutoApprove IntString `json:"auto_approve"`
			BodySrc     string    `json:"body_src"`
			BodyTgt     string    `json:"body_tgt"`
			CallbackURL string    `json:"callback_url"`
			CaptchaURL  string    `json:"captcha_url"`
			Credits     FloatString
			Ctime       int64
			Currency    string
			ETA         int
			JobId       IntString `json:"job_id"`
			LcSrc       string    `json:"lc_src"`
			LcTgt       string    `json:"lc_tgt"`
			Mt          int
			PreviewURL  string `json:"preview_url"`
			Slug        IntString
			Status      string
			Tier        string
			UnitCount   IntString `json:"unit_count"`
		}
	}
	Err *FailedResponse
}

func (gengo *Gengo) Job(jobId int, optionalParams map[string]string) (r *JobResponse, err error) {
	method := fmt.Sprintf("translate/job/%d", jobId)
	b := getRequest(method, *gengo, true, optionalParams)
	fmt.Println(string(b))
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return nil, err
	}
	return
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

func (gengo *Gengo) ReviseJob(jobId int, reviseAction ReviseAction) (err error) {
	method := fmt.Sprintf("translate/job/%d", jobId)
	reviseActionJSON, err := json.Marshal(reviseAction)
	if err != nil {
		return err
	}
	b := putRequest(method, *gengo, string(reviseActionJSON))
	var r EmptyResponse
	err = json.Unmarshal(b, &r)
	if err != nil {
		return err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return err
	}
	return
}

type ApproveAction struct {
	ActionType    string  `json:"action"`
	Rating        *int    `json:"rating,omitempty"`
	ForTranslator *string `json:"for_translator,omitempty"`
	ForGengo    *string `json:"for_gengo,omitempty"`
	Public        *int    `json:"public,omitempty"`
}

func NewApproveAction() (approveAction ApproveAction) {
	approveAction = ApproveAction{ActionType: "approve"}
	return
}

func (approveAction *ApproveAction) AddRating(rating int) {
	approveAction.Rating = &rating
}

func (approveAction *ApproveAction) AddForTranslator(forTranslator string) {
	approveAction.ForTranslator = &forTranslator
}

func (approveAction *ApproveAction) AddForGengo(forGengo string) {
	approveAction.ForGengo = &forGengo
}

func (approveAction *ApproveAction) AddPublic(public int) {
	approveAction.Public = &public
}

func (gengo *Gengo) ApproveJob(jobId int, approveAction ApproveAction) (err error) {
	method := fmt.Sprintf("translate/job/%d", jobId)
	approveActionJSON, err := json.Marshal(approveAction)
	if err != nil {
		log.Fatal(err)
	}
	b := putRequest(method, *gengo, string(approveActionJSON))
	var r EmptyResponse
	err = json.Unmarshal(b, &r)
	if err != nil {
		return err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return err
	}
	return
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

func (rejectAction *RejectAction) AddFollowUp(followUp string) {
	rejectAction.FollowUp = &followUp
}

func (gengo *Gengo) RejectJob(jobId int, rejectAction RejectAction) (err error) {
	method := fmt.Sprintf("translate/job/%d", jobId)
	rejectActionJSON, err := json.Marshal(rejectAction)
	if err != nil {
		log.Fatal(err)
	}
	b := putRequest(method, *gengo, string(rejectActionJSON))
	var r EmptyResponse
	err = json.Unmarshal(b, &r)
	if err != nil {
		return err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return err
	}
	return
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

func (jobPayload *JobPayload) AddForce(force int) {
	jobPayload.Force = &force
}

func (jobPayload *JobPayload) AddComment(comment string) {
	jobPayload.Comment = &comment
}

func (jobPayload *JobPayload) AddUsePreferred(usePreferred int) {
	jobPayload.UsePreferred = &usePreferred
}

func (jobPayload *JobPayload) AddCallbackURL(callbackURL string) {
	jobPayload.CallbackURL = &callbackURL
}

func (jobPayload *JobPayload) AddAutoApprove(autoApprove int) {
	jobPayload.AutoApprove = &autoApprove
}

func (jobPayload *JobPayload) AddCustomData(customData string) {
	jobPayload.CustomData = &customData
}

func (gengo *Gengo) PostJob(jobPayload JobPayload) (r *JobResponse, err error) {
	type Job struct {
		JobPayload JobPayload `json:"job"`
	}
	method := "translate/job"
	job := Job{JobPayload: jobPayload}
	postJobJSON, err := json.Marshal(job)
	if err != nil {
		log.Fatal(err)
	}
	b := postRequest(method, *gengo, string(postJobJSON))
	fmt.Println(string(b))
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return nil, err
	}
	return
}

func (gengo *Gengo) JobsGroup(groupId int) interface{} {
	method := fmt.Sprintf("translate/jobs/group/%d", groupId)
	return getRequest(method, *gengo, true, nil)
}

func (gengo *Gengo) Jobs(optionalParams map[string]string) interface{} {
	method := "translate/jobs"
	return getRequest(method, *gengo, true, optionalParams)
}

func (gengo *Gengo) JobsByIds(jobIds []int) interface{} {
	jobIdsStrings := []string{}
	for _, jobId := range jobIds {
		jobIdsStrings = append(jobIdsStrings, strconv.Itoa(jobId))
	}
	jobIdsString := strings.Join(jobIdsStrings, ",")
	method := fmt.Sprintf("translate/jobs/%s", jobIdsString)
	return getRequest(method, *gengo, true, nil)
}

type JobArray struct {
	Jobs    []JobPayload `json:"jobs"`
	AsGroup *int         `json:"as_group,omitempty"`
}

func (jobArray *JobArray) AddAsGroup(asGroup int) {
	jobArray.AsGroup = &asGroup
}

func NewJobArray(jobs []JobPayload) (jobArray JobArray) {
	jobArray = JobArray{Jobs: jobs}
	return
}

func (gengo *Gengo) PostJobs(jobArray JobArray) interface{} {
	method := "translate/jobs"
	postJobsJSON, err := json.Marshal(jobArray)
	if err != nil {
		log.Fatal(err)
	}
	return postRequest(method, *gengo, string(postJobsJSON))
}

type LanguagePairsResponse struct {
	Opstat   string
	Response []struct {
		Currency  string
		LcSrc     string `json:"lc_src"`
		LcTgt     string `json:"lc_tgt"`
		Tier      string
		UnitPrice FloatString `json:"unit_price"`
	}
	Err *FailedResponse
}

func (gengo *Gengo) LanguagePairs(optionalParams map[string]string) (r *LanguagePairsResponse, err error) {
	method := "translate/service/language_pairs"
	b := getRequest(method, *gengo, false, optionalParams)
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return nil, err
	}
	return
}

type LanguagesResponse struct {
	Opstat   string
	Response []struct {
		Language      string
		Lc            string
		LocalizedName string `json:"localized_name"`
		UnitType      string `json:"unit_type"`
	}
	Err *FailedResponse
}

func (gengo *Gengo) Languages() (r *LanguagesResponse, err error) {
	method := "translate/service/languages"
	b := getRequest(method, *gengo, false, nil)
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return nil, err
	}
	return
}

type JobsQuoteResponse struct {
	Opstat   string
	Response struct {
		Jobs []struct {
			UnitCount int `json:"unit_count"`
			Credits   float64
			ETA       int
			Currency  string
		}
	}
    Err *FailedResponse `json:"err,omitempty"`
}

func (gengo *Gengo) JobsQuote(jobArray JobArray) (r *JobsQuoteResponse, err error) {
	method := "translate/service/quote"
	jobsQuoteJSON, err := json.Marshal(jobArray)
	if err != nil {
		log.Fatal(err)
	}
	b := postRequest(method, *gengo, string(jobsQuoteJSON))
	err = json.Unmarshal(b, &r)
	if err != nil {
		return nil, err
	}
	if r.Opstat == "error" {
		e := fmt.Sprintf("Failed response.  Code: %d, Message: %s", r.Err.Code, r.Err.Msg)
		err = errors.New(e)
		return nil, err
	}
	return
}
