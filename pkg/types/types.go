package types

import "github.com/blocto/solana-go-sdk/common"

type ConfigStruct struct {
	RpcURL string `json:"rpc_url"`
}

type AccountData struct {
	AccountAddress  common.PublicKey
	AccountMnemonic string
	AccountKey      string
	LogData         string
}
