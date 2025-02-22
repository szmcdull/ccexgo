package test_account

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/szmcdull/ccexgo/exchange/ftx"
)

func TestAccountLeverage(t *testing.T) {
	key := os.Getenv("F_KEY")
	secret := os.Getenv("F_SECRET")
	if len(key) == 0 || len(secret) == 0 {
		t.Fatalf("missing env key, secret")
	}

	client := ftx.NewRestClient(key, secret)
	ctx := context.Background()
	pos, err := client.Positions(ctx)
	if err != nil {
		t.Fatalf("got leverage fail %s", err.Error())
	}

	fmt.Printf("got leverage %v\n", pos)
}
