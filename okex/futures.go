package okex

import (
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/nntaoli-project/goex"
	"sort"
	"strconv"
	"strings"
	"time"
)

type FuturesWs struct {
	v3Ws           *baseWs
	tickerCallback func(*FutureTicker)
	depthCallback  func(*Depth)
	tradeCallback  func(*Trade, string)
	klineCallback  func(*FutureKline, int, string)
}

func NewFuturesWs() *FuturesWs {
	ws := &FuturesWs{}
	ws.v3Ws = NewOKExV3Ws(ws.handle)
	return ws
}

func (ws *FuturesWs) TickerCallback(tickerCallback func(*FutureTicker)) {
	ws.tickerCallback = tickerCallback
}

func (ws *FuturesWs) DepthCallback(depthCallback func(*Depth)) {
	ws.depthCallback = depthCallback
}

func (ws *FuturesWs) TradeCallback(tradeCallback func(*Trade, string)) {
	ws.tradeCallback = tradeCallback
}

func (ws *FuturesWs) KlineCallback(klineCallback func(*FutureKline, int, string)) {
	ws.klineCallback = klineCallback
}

func (ws *FuturesWs) SetCallbacks(tickerCallback func(*FutureTicker),
	depthCallback func(*Depth),
	tradeCallback func(*Trade, string),
	klineCallback func(*FutureKline, int, string)) {
	ws.tickerCallback = tickerCallback
	ws.depthCallback = depthCallback
	ws.tradeCallback = tradeCallback
	ws.klineCallback = klineCallback
}

func (ws *FuturesWs) getChannelName(currencyPair CurrencyPair, contractType string) string {
	var (
		prefix      string
		contractId  string
		channelName string
	)

	if contractType == SWAP_CONTRACT {
		prefix = "swap"
		contractId = fmt.Sprintf("%s-SWAP", currencyPair.ToSymbol("-"))
	} else {
		prefix = "futures"
		//contractId = ws.base.OKExFuture.GetFutureContractId(currencyPair, contractType)
		//logger.Info("contractid=", contractId)
	}

	channelName = prefix + "/%s:" + contractId

	return channelName
}

func (ws *FuturesWs) SubscribeDepth(pair CurrencyPair, size int, contract string) error {
	if ws.depthCallback == nil {
		return errors.New("please set depth callback func")
	}

	chName := ws.getChannelName(pair, contract)

	return ws.v3Ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{fmt.Sprintf(chName, "depth5")}})
}

func (ws *FuturesWs) SubscribeTicker(currencyPair CurrencyPair, contractType string) error {
	if ws.tickerCallback == nil {
		return errors.New("please set ticker callback func")
	}
	chName := ws.getChannelName(currencyPair, contractType)
	return ws.v3Ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{fmt.Sprintf(chName, "ticker")}})
}

func (ws *FuturesWs) SubscribeTrade(currencyPair CurrencyPair, contractType string) error {
	if ws.tradeCallback == nil {
		return errors.New("please set trade callback func")
	}
	chName := ws.getChannelName(currencyPair, contractType)
	return ws.v3Ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{fmt.Sprintf(chName, "trade")}})
}

func (ws *FuturesWs) SubscribeKline(currencyPair CurrencyPair, period int, contractType string) error {
	if ws.klineCallback == nil {
		return errors.New("place set kline callback func")
	}

	seconds := adaptKLinePeriod(period)
	if seconds == -1 {
		return fmt.Errorf("unsupported kline period %d in okex", period)
	}

	chName := ws.getChannelName(currencyPair, contractType)
	return ws.v3Ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{fmt.Sprintf(chName, fmt.Sprintf("candle%ds", seconds))}})
}

func (ws *FuturesWs) getContractAliasAndCurrencyPairFromInstrumentId(instrumentId string) (alias string, pair CurrencyPair) {
	ar := strings.Split(instrumentId, "-")
	return instrumentId, NewCurrencyPair2(fmt.Sprintf("%s_%s", ar[0], ar[1]))
}

func (ws *FuturesWs) handle(channel string, data json.RawMessage) error {
	var (
		err           error
		ch            string
		tickers       []tickerResponse
		depthResp     []depthResponse
		dep           Depth
		tradeResponse []struct {
			Side         string  `json:"side"`
			TradeId      int64   `json:"trade_id,string"`
			Price        float64 `json:"price,string"`
			Qty          float64 `json:"qty,string"`
			InstrumentId string  `json:"instrument_id"`
			Timestamp    string  `json:"timestamp"`
		}
		klineResponse []struct {
			Candle       []string `json:"candle"`
			InstrumentId string   `json:"instrument_id"`
		}
	)

	if strings.Contains(channel, "futures/candle") {
		ch = "candle"
	} else {
		ch, err = ws.v3Ws.parseChannel(channel)
		if err != nil {
			//logger.Errorf("[%s] parse channel err=%s ,  originChannel=%s", ws.base.GetExchangeName(), err, ch)
			return nil
		}
	}

	switch ch {
	case "ticker":
		err = json.Unmarshal(data, &tickers)
		if err != nil {
			return err
		}

		for _, t := range tickers {
			alias, pair := ws.getContractAliasAndCurrencyPairFromInstrumentId(t.InstrumentId)
			date, _ := time.Parse(time.RFC3339, t.Timestamp)
			ws.tickerCallback(&FutureTicker{
				Ticker: &Ticker{
					Pair: pair,
					Last: t.Last,
					Buy:  t.BestBid,
					Sell: t.BestAsk,
					High: t.High24h,
					Low:  t.Low24h,
					Vol:  t.Volume24h,
					Date: uint64(date.UnixNano() / int64(time.Millisecond)),
				},
				ContractType: alias,
			})
		}
		return nil
	case "candle":
		err = json.Unmarshal(data, &klineResponse)
		if err != nil {
			return err
		}

		for _, t := range klineResponse {
			ali, pair := ws.getContractAliasAndCurrencyPairFromInstrumentId(t.InstrumentId)
			ts, _ := time.Parse(time.RFC3339, t.Candle[0])
			//granularity := adaptKLinePeriod(KlinePeriod(period))
			ws.klineCallback(&FutureKline{
				Kline: &Kline{
					Pair:      pair,
					High:      ToFloat64(t.Candle[2]),
					Low:       ToFloat64(t.Candle[3]),
					Timestamp: ts.Unix(),
					Open:      ToFloat64(t.Candle[1]),
					Close:     ToFloat64(t.Candle[4]),
					Vol:       ToFloat64(t.Candle[5]),
				},
				Vol2: ToFloat64(t.Candle[6]),
			}, 1, ali)
		}
		return nil
	case "depth5":
		err := json.Unmarshal(data, &depthResp)
		if err != nil {
			//logger.Error(err)
			return err
		}
		if len(depthResp) == 0 {
			return nil
		}
		alias, pair := ws.getContractAliasAndCurrencyPairFromInstrumentId(depthResp[0].InstrumentId)
		dep.Pair = pair
		dep.ContractType = alias
		dep.UTime, _ = time.Parse(time.RFC3339, depthResp[0].Timestamp)
		for _, itm := range depthResp[0].Asks {
			dep.AskList = append(dep.AskList, DepthRecord{
				Price:  ToFloat64(itm[0]),
				Amount: ToFloat64(itm[1])})
		}
		for _, itm := range depthResp[0].Bids {
			dep.BidList = append(dep.BidList, DepthRecord{
				Price:  ToFloat64(itm[0]),
				Amount: ToFloat64(itm[1])})
		}
		sort.Sort(sort.Reverse(dep.AskList))
		//call back func
		ws.depthCallback(&dep)
		return nil
	case "trade":
		err := json.Unmarshal(data, &tradeResponse)
		if err != nil {
			//logger.Error("unmarshal error :", err)
			return err
		}

		for _, resp := range tradeResponse {
			alias, pair := ws.getContractAliasAndCurrencyPairFromInstrumentId(resp.InstrumentId)

			tradeSide := SELL
			switch resp.Side {
			case "buy":
				tradeSide = BUY
			}

			t, err := time.Parse(time.RFC3339, resp.Timestamp)
			if err != nil {
				//logger.Warn("parse timestamp error:", err)
			}

			ws.tradeCallback(&Trade{
				Tid:    resp.TradeId,
				Type:   tradeSide,
				Amount: resp.Qty,
				Price:  resp.Price,
				Date:   t.Unix(),
				Pair:   pair,
			}, alias)
		}
		return nil
	}

	return fmt.Errorf("[%s] unknown websocket message: %s", ch, string(data))
}

func (ws *FuturesWs) getKlinePeriodFormChannel(channel string) int {
	metas := strings.Split(channel, ":")
	if len(metas) != 2 {
		return 0
	}
	i, _ := strconv.ParseInt(metas[1], 10, 64)
	return int(i)
}
