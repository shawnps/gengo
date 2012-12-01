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
		t.Errorf("Account stats Opstat was not ok: %s", rsp)
	}
}

func TestAccountBalance(t *testing.T) {
	rsp, err := gengo.AccountBalance()
	if err != nil {
		t.Errorf(err.Error())
	}
	if rsp.Opstat != "ok" {
		t.Errorf("Account balance Opstat was not ok: %s", rsp)
	}
}
