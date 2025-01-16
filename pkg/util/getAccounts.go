package util

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha512"
	"errors"
	"fmt"
	"github.com/blocto/solana-go-sdk/common"
	"github.com/blocto/solana-go-sdk/types"
	"github.com/mr-tron/base58"
	log "github.com/sirupsen/logrus"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/pbkdf2"
	types2 "main/pkg/types"
	"math/big"
)

func derive(key []byte, chainCode []byte, segment uint32) ([]byte, []byte) {
	buf := []byte{0}
	buf = append(buf, key...)
	buf = append(buf, big.NewInt(int64(segment)).Bytes()...)

	h := hmac.New(sha512.New, chainCode)
	h.Write(buf)
	I := h.Sum(nil)

	IL := I[:32]
	IR := I[32:]

	return IL, IR
}

func checkMnemonic(target string) (types2.AccountData, error) {
	if !bip39.IsMnemonicValid(target) {
		return types2.AccountData{}, errors.New("invalid mnemonic")
	}

	seed := pbkdf2.Key([]byte(target), []byte("mnemonic"), 2048, 64, sha512.New)

	h := hmac.New(sha512.New, []byte("ed25519 seed"))
	h.Write(seed)
	sum := h.Sum(nil)

	key := sum[:32]
	chainCode := sum[32:]

	segments := []uint32{
		0x80000000 + 44,
		0x80000000 + 501,
		0x80000000 + 0,
		0x80000000 + 0,
	}

	for _, segment := range segments {
		buf := []byte{0}
		buf = append(buf, key...)
		buf = append(buf, big.NewInt(int64(segment)).Bytes()...)

		// Derive new key
		h = hmac.New(sha512.New, chainCode)
		h.Write(buf)
		I := h.Sum(nil)

		key = I[:32]
		chainCode = I[32:]
	}

	key = ed25519.NewKeyFromSeed(key)
	wallet, err := types.AccountFromBase58(base58.Encode(key))
	if err != nil {
		return types2.AccountData{}, err
	}

	return types2.AccountData{
		AccountAddress:  wallet.PublicKey,
		AccountMnemonic: target,
		AccountKey:      base58.Encode(key),
		LogData:         wallet.PublicKey.String(),
	}, nil
}

func checkPKey(target string) (types2.AccountData, error) {
	wallet, err := types.AccountFromBase58(target)
	if err != nil {
		return types2.AccountData{}, err
	}

	return types2.AccountData{
		AccountAddress:  wallet.PublicKey,
		AccountMnemonic: "",
		AccountKey:      target,
		LogData:         wallet.PublicKey.String(),
	}, nil
}

func checkAddress(address string) (types2.AccountData, error) {
	decoded, err := base58.Decode(address)
	if err != nil {
		return types2.AccountData{}, fmt.Errorf("invalid address")
	}

	if len(decoded) != 32 {
		return types2.AccountData{}, fmt.Errorf("invalid address")
	}

	return types2.AccountData{
		AccountAddress:  common.PublicKeyFromString(address),
		AccountMnemonic: "",
		AccountKey:      "",
		LogData:         address,
	}, nil
}

func GetAccounts(accountsData []string) []types2.AccountData {
	var result []types2.AccountData

	for _, accountData := range accountsData {
		formattedAccountData, err := checkMnemonic(accountData)

		if err == nil {
			result = append(result, formattedAccountData)
			continue
		}

		formattedAccountData, err = checkPKey(accountData)

		if err == nil {
			result = append(result, formattedAccountData)
			continue
		}

		formattedAccountData, err = checkAddress(accountData)

		if err == nil {
			result = append(result, formattedAccountData)
			continue
		}

		log.Printf("%s | Not Mnemonic/PKey/Address", accountData)
	}

	return result
}
