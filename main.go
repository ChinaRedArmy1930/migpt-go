package main

import (
	"context"

	_ "migpt-go/internal/log"
	"migpt-go/mi/base/account"
)

func init() {

}

func main() {
	account.InitMIAccount(context.Background())
}
