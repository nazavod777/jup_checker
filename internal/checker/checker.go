package checker

import (
	"encoding/json"
	"fmt"
	"github.com/blocto/solana-go-sdk/common"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"main/pkg/global"
	"main/pkg/types"
	"main/pkg/util"
	"strings"
)

func getAllocation(
	accountData types.AccountData,
) float64 {
	var err error

	for {
		client := util.GetClient(util.GetProxy())

		req := fasthttp.AcquireRequest()

		req.SetRequestURI(fmt.Sprintf("https://jupuary.jup.ag/api/allocation?wallet=%s",
			accountData.AccountAddress))
		req.Header.SetMethod("GET")
		req.Header.Set("accept", "*/*")
		req.Header.Set("accept-language", "ru,en;q=0.9")
		req.Header.Set("pragma", "no-cache")
		req.Header.SetReferer("https://jupuary.jup.ag/allocation")
		req.Header.SetUserAgent("Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36")

		resp := fasthttp.AcquireResponse()

		if err = client.Do(req, resp); err != nil {
			log.Printf("%s | Error When Doing Request When Parsing Allocation: %s", accountData.LogData, err)

			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			continue
		}

		var response map[string]interface{}

		if err := json.Unmarshal(resp.Body(), &response); err != nil {
			log.Printf("%s | Failed To Parse JSON Response When Parsing Balance: %s, response: %s",
				accountData.LogData, err, string(resp.Body()))
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			continue
		}

		data, ok := response["data"]
		status, statusOk := response["status"]

		if !ok || !statusOk {
			log.Printf("%s | Wrong Response When Parsing Allocation: %s",
				accountData.LogData, string(resp.Body()))
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			continue
		}

		if statusStr, ok := status.(string); ok && statusStr == "success" {
			if data == nil {
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(resp)
				return 0
			}
		}

		dataMap, ok := data.(map[string]interface{})
		if !ok {
			log.Printf("%s | Wrong Response When Parsing Allocation: %s",
				accountData.LogData, string(resp.Body()))
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			continue
		}

		if value, ok := dataMap["total_allocated"]; ok {
			switch v := value.(type) {
			case float64:
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(resp)
				return v
			case nil:
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(resp)
				return 0
			default:
				log.Printf("%s | Wrong Response When Parsing Allocation: %s",
					accountData.LogData, string(resp.Body()))
				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(resp)
				continue
			}
		} else {
			log.Printf("%s | Wrong Response When Parsing Allocation: %s",
				accountData.LogData, string(resp.Body()))
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			continue
		}
	}
}

func getAccountOwner(
	accountData types.AccountData,
	targetAddress string,
) string {
	type responseStruct struct {
		Jsonrpc string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  *struct {
			Value *struct {
				Data *struct {
					Parsed *struct {
						Info *struct {
							Owner string `json:"owner"`
						} `json:"info"`
					} `json:"parsed"`
				} `json:"data"`
			} `json:"value"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}

	params := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getAccountInfo",
		"params": []interface{}{
			targetAddress,
			map[string]interface{}{
				"encoding": "jsonParsed",
			},
		},
	}

	body, err := json.Marshal(params)

	if err != nil {
		log.Panicf("%s | Error When Marshalling JSON When Getting Account Info: %v",
			accountData.LogData, err)
	}

	for {
		client := util.GetClient(util.GetProxy())

		req := fasthttp.AcquireRequest()

		req.SetRequestURI(global.Config.RpcURL)
		req.Header.SetMethod("POST")
		req.Header.SetContentType("application/json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("accept", "*/*")
		req.Header.Set("accept-language", "ru,en;q=0.9,vi;q=0.8,es;q=0.7,cy;q=0.6")
		req.SetBody(body)

		resp := fasthttp.AcquireResponse()

		if err = client.Do(req, resp); err != nil {
			log.Printf("%s | Error When Doing Request When Getting Account Info: %s", accountData.LogData, err)

			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			continue
		}

		if resp.StatusCode() != fasthttp.StatusOK {
			log.Printf("%s | Wrong Status Code When Getting Account Info: %d",
				accountData.LogData, resp.StatusCode())

			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			continue
		}

		var response responseStruct

		if err = json.Unmarshal(resp.Body(), &response); err != nil {
			log.Printf("%s | Failed To Parse JSON Response When Logging: %s, response: %s",
				accountData.LogData, err, string(resp.Body()))
			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			continue
		}

		if response.Error != nil {
			log.Printf("%s | Wrong Response When Getting Account Info: %s",
				accountData.LogData, string(resp.Body()))

			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
			continue
		}

		fasthttp.ReleaseRequest(req)
		fasthttp.ReleaseResponse(resp)

		if response.Result != nil &&
			response.Result.Value != nil &&
			response.Result.Value.Data.Parsed.Info.Owner != "" {
			return response.Result.Value.Data.Parsed.Info.Owner
		}

		return ""
	}
}

func CheckAccount(accountData types.AccountData) {
	totalAllocation := getAllocation(accountData)

	if totalAllocation <= 0 {
		log.Printf("%s | Not Eligible", accountData.LogData)
		return
	}

	mintAddress := common.PublicKeyFromString("JUPyiwrYJFskUPiHa7hkeR8VUtAeFoSYbKedZNsDvCN")
	associatedTokenProgram := common.SPLAssociatedTokenAccountProgramID
	splTokenProgram := common.TokenProgramID

	ata, _, err := common.FindProgramAddress(
		[][]byte{
			accountData.AccountAddress.Bytes(),
			splTokenProgram.Bytes(),
			mintAddress.Bytes(),
		},
		associatedTokenProgram,
	)
	if err != nil {
		log.Fatalf("failed to calculate ATA: %v", err)
	}

	tokenAccountAddress := ata.ToBase58()
	tokenAccountOwner := getAccountOwner(accountData, tokenAccountAddress)

	var resultData string

	if accountData.AccountMnemonic != "" {
		resultData = accountData.AccountMnemonic
	} else if accountData.AccountKey != "" {
		resultData = accountData.AccountKey
	} else {
		resultData = accountData.AccountAddress.String()
	}

	if strings.ToLower(tokenAccountOwner) == strings.ToLower(accountData.AccountAddress.String()) {
		log.Printf("%s | Total Alloaction: %g $JUP | Not Chaned Authority", accountData.LogData, totalAllocation)

		util.AppendFile("without_authority.txt",
			fmt.Sprintf("%s | %g $JUP\n", resultData, totalAllocation))
	} else {
		log.Printf("%s | Total Alloaction: %g $JUP | Changed Authority", accountData.LogData, totalAllocation)

		util.AppendFile("with_authority.txt",
			fmt.Sprintf("%s | %g $JUP\n", resultData, totalAllocation))
	}
}
