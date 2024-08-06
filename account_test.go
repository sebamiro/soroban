package soroban_test

import (
	"testing"

	"github.com/sebamiro/soroban"
)

func TestGetAccount(t *testing.T) {
	sorobanClient := soroban.Client{}
	sorobanClient.URL = LOCAL_NETWORK
	sorobanClient.PassPhrase = LOCAL_PASSPHRASE

	a, err := sorobanClient.GetAccountEntry("GDDFXO5LE6JLE7E4HYN7EWBDJSKJ3NV7MAC4UN7LY7BUSD6JNPUAUK4K")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(a)
}
