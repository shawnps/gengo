package gengo

import (
	"os"
	"testing"
)

var (
	pubKey  = os.Getenv("GENGO_PUBKEY")
	privKey = os.Getenv("GENGO_PRIVKEY")
	gengo   = Gengo{pubKey, privKey, true}
)

func TestAccountStats(t *testing.T) {
	rsp, err := gengo.AccountStats()
	if err != nil {
		t.Errorf(err.Error())
	}
	if rsp.Opstat != "ok" {
		t.Errorf("Account stats opstat was not ok: %s", rsp)
	}
}

func TestAccountBalance(t *testing.T) {
	rsp, err := gengo.AccountBalance()
	if err != nil {
		t.Errorf(err.Error())
	}
	if rsp.Opstat != "ok" {
		t.Errorf("Account balance opstat was not ok: %s", rsp)
	}
}

func TestLanguagePairs(t *testing.T) {
	rsp, err := gengo.LanguagePairs()
	if err != nil {
		t.Errorf(err.Error())
	}
	if rsp.Opstat != "ok" {
		t.Errorf("Language pairs opstat was not ok: %s", rsp)
	}
}

func TestLanguages(t *testing.T) {
	rsp, err := gengo.Languages()
	if err != nil {
		t.Errorf(err.Error())
	}
	if rsp.Opstat != "ok" {
		t.Errorf("Languages opstat was not ok: %s", rsp)
	}
}
