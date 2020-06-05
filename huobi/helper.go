package huobi

import (
	json2 "encoding/json"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/nntaoli-project/goex"
	"sort"
	"strings"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type DepthResponse struct {
	Bids [][]float64
	Asks [][]float64
	Ts   int64 `json:"ts"`
}

type WsResponse struct {
	Ch   string
	Ts   int64
	Tick json2.RawMessage
}

type TradeResponse struct {
	Id   int64
	Ts   int64
	Data []struct {
		Id        int64
		Amount    float64
		Price     float64
		Direction string
		Ts        int64
	}
}

type DetailResponse struct {
	Id     int64
	Open   float64
	Close  float64
	High   float64
	Low    float64
	Amount float64
	Vol    float64
	Count  int64
}

func ParseDepthFromResponse(r DepthResponse) goex.Depth {
	var dep goex.Depth
	for _, bid := range r.Bids {
		dep.BidList = append(dep.BidList, goex.DepthRecord{Price: bid[0], Amount: bid[1]})
	}

	for _, ask := range r.Asks {
		dep.AskList = append(dep.AskList, goex.DepthRecord{Price: ask[0], Amount: ask[1]})
	}

	sort.Sort(sort.Reverse(dep.BidList))
	sort.Sort(sort.Reverse(dep.AskList))
	return dep
}

func ParseCurrencyPairFromSpotWsCh(ch string) goex.CurrencyPair {
	meta := strings.Split(ch, ".")
	if len(meta) < 2 {
		//logger.Errorf("parse error, ch=%s", ch)
		return goex.UNKNOWN_PAIR
	}

	currencyPairStr := meta[1]
	if strings.HasSuffix(currencyPairStr, "usdt") {
		currencyA := strings.TrimSuffix(currencyPairStr, "usdt")
		return goex.NewCurrencyPair2(fmt.Sprintf("%s_usdt", currencyA))
	}

	if strings.HasSuffix(currencyPairStr, "btc") {
		currencyA := strings.TrimSuffix(currencyPairStr, "btc")
		return goex.NewCurrencyPair2(fmt.Sprintf("%s_btc", currencyA))
	}

	if strings.HasSuffix(currencyPairStr, "eth") {
		currencyA := strings.TrimSuffix(currencyPairStr, "eth")
		return goex.NewCurrencyPair2(fmt.Sprintf("%s_eth", currencyA))
	}

	if strings.HasSuffix(currencyPairStr, "husd") {
		currencyA := strings.TrimSuffix(currencyPairStr, "husd")
		return goex.NewCurrencyPair2(fmt.Sprintf("%s_husd", currencyA))
	}

	if strings.HasSuffix(currencyPairStr, "ht") {
		currencyA := strings.TrimSuffix(currencyPairStr, "ht")
		return goex.NewCurrencyPair2(fmt.Sprintf("%s_ht", currencyA))
	}

	if strings.HasSuffix(currencyPairStr, "trx") {
		currencyA := strings.TrimSuffix(currencyPairStr, "trx")
		return goex.NewCurrencyPair2(fmt.Sprintf("%s_trx", currencyA))
	}

	return goex.UNKNOWN_PAIR
}
