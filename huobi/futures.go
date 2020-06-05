package huobi

import (
	"bytes"
	"errors"
	"fmt"
	. "github.com/nntaoli-project/goex"
	"strings"
	"sync"
	"time"
)

type FuturesWs struct {
	*WsBuilder
	sync.Once
	wsConn *WsConn

	tickerCallback func(*FutureTicker)
	depthCallback  func(*Depth)
	tradeCallback  func(*Trade, string)
	klineCallback  func(*FutureKline, int, string)
}

func NewFutureWs() *FuturesWs {
	ws := &FuturesWs{WsBuilder: NewWsBuilder()}
	ws.WsBuilder = ws.WsBuilder.
		WsUrl("wss://api.hbdm.com/ws").
		AutoReconnect().
		//Heartbeat([]byte("{\"event\": \"ping\"} "), 30*time.Second).
		//Heartbeat(func() []byte { return []byte("{\"op\":\"ping\"}") }(), 5*time.Second).
		DecompressFunc(GzipDecompress).
		ProtoHandleFunc(ws.handle)
	return ws
}

func (ws *FuturesWs) SetCallbacks(tickerCallback func(*FutureTicker),
	depthCallback func(*Depth),
	tradeCallback func(*Trade, string)) {
	ws.tickerCallback = tickerCallback
	ws.depthCallback = depthCallback
	ws.tradeCallback = tradeCallback
}

func (ws *FuturesWs) TickerCallback(call func(ticker *FutureTicker)) {
	ws.tickerCallback = call
}

func (ws *FuturesWs) TradeCallback(call func(trade *Trade, contract string)) {
	ws.tradeCallback = call
}

func (ws *FuturesWs) DepthCallback(call func(depth *Depth)) {
	ws.depthCallback = call
}

func (ws *FuturesWs) KlineCallback(call func(*FutureKline, int, string)) {
	ws.klineCallback = call
}
func (ws *FuturesWs) SubscribeTicker(pair CurrencyPair, contract string) error {
	if ws.tickerCallback == nil {
		return errors.New("please set ticker callback func")
	}
	return ws.subscribe(map[string]interface{}{
		"id":  "ticker_1",
		"sub": fmt.Sprintf("market.%s_%s.detail", pair.CurrencyA.Symbol, ws.adaptContractSymbol(contract))})
}

func (ws *FuturesWs) SubscribeDepth(pair CurrencyPair, size int, contract string) error {
	if ws.depthCallback == nil {
		return errors.New("please set depth callback func")
	}
	return ws.subscribe(map[string]interface{}{
		"id":  "futures.depth",
		"sub": fmt.Sprintf("market.%s_%s.depth.size_20.high_freq", pair.CurrencyA.Symbol, ws.adaptContractSymbol(contract))})
}

func (ws *FuturesWs) SubscribeKline(pair CurrencyPair, period int, contractType string) error {
	return nil
}

func (ws *FuturesWs) SubscribeTrade(pair CurrencyPair, contract string) error {
	if ws.tradeCallback == nil {
		return errors.New("please set trade callback func")
	}
	return ws.subscribe(map[string]interface{}{
		"id":  "trade_3",
		"sub": fmt.Sprintf("market.%s_%s.trade.detail", pair.CurrencyA.Symbol, ws.adaptContractSymbol(contract))})
}

func (ws *FuturesWs) subscribe(sub map[string]interface{}) error {
	//	log.Println(sub)
	ws.connectWs()
	return ws.wsConn.Subscribe(sub)
}

func (ws *FuturesWs) connectWs() {
	ws.Do(func() {
		ws.wsConn = ws.WsBuilder.Build()
	})
}

func (ws *FuturesWs) handle(msg []byte) error {
	//心跳
	if bytes.Contains(msg, []byte("ping")) {
		pong := bytes.ReplaceAll(msg, []byte("ping"), []byte("pong"))
		ws.wsConn.SendMessage(pong)
		return nil
	}

	var resp WsResponse
	err := json.Unmarshal(msg, &resp)
	if err != nil {
		return err
	}

	if resp.Ch == "" {
		//logger.Warnf("[%s] ch == \"\" , msg=%s", ws.wsConn.WsUrl, string(msg))
		return nil
	}

	pair, contract, err := ws.parseCurrencyAndContract(resp.Ch)
	if err != nil {
		//logger.Errorf("[%s] parse currency and contract err=%s", ws.wsConn.WsUrl, err)
		return err
	}

	if strings.Contains(resp.Ch, ".depth.") {
		var depResp DepthResponse
		err := json.Unmarshal(resp.Tick, &depResp)
		if err != nil {
			return err
		}

		dep := ParseDepthFromResponse(depResp)
		dep.ContractType = contract
		dep.Pair = pair
		dep.UTime = time.Unix(0, resp.Ts*int64(time.Millisecond))

		ws.depthCallback(&dep)
		return nil
	}

	if strings.HasSuffix(resp.Ch, "trade.detail") {
		var tradeResp TradeResponse
		err := json.Unmarshal(resp.Tick, &tradeResp)
		if err != nil {
			return err
		}
		trades := ws.parseTrade(tradeResp)
		for _, v := range trades {
			v.Pair = pair
			ws.tradeCallback(&v, contract)
		}
		return nil
	}

	if strings.HasSuffix(resp.Ch, ".detail") {
		var detail DetailResponse
		err := json.Unmarshal(resp.Tick, &detail)
		if err != nil {
			return err
		}
		ticker := ws.parseTicker(detail)
		ticker.ContractType = contract
		ticker.Pair = pair
		ws.tickerCallback(&ticker)
		return nil
	}

	//logger.Errorf("[%s] unknown message, msg=%s", ws.wsConn.WsUrl, string(msg))

	return nil
}

func (ws *FuturesWs) parseTicker(r DetailResponse) FutureTicker {
	return FutureTicker{Ticker: &Ticker{High: r.High, Low: r.Low, Vol: r.Amount}}
}

func (ws *FuturesWs) parseCurrencyAndContract(ch string) (CurrencyPair, string, error) {
	el := strings.Split(ch, ".")
	if len(el) < 2 {
		return UNKNOWN_PAIR, "", errors.New(ch)
	}
	cs := strings.Split(el[1], "_")
	contract := ""
	switch cs[1] {
	case "CQ":
		contract = QUARTER_CONTRACT
	case "NW":
		contract = NEXT_WEEK_CONTRACT
	case "CW":
		contract = THIS_WEEK_CONTRACT
	}
	return NewCurrencyPair(NewCurrency(cs[0], ""), USD), contract, nil
}

func (ws *FuturesWs) parseTrade(r TradeResponse) []Trade {
	var trades []Trade
	for _, v := range r.Data {
		trades = append(trades, Trade{
			Tid:    v.Id,
			Price:  v.Price,
			Amount: v.Amount,
			Type:   AdaptTradeSide(v.Direction),
			Date:   v.Ts})
	}
	return trades
}

func (ws *FuturesWs) adaptContractSymbol(contract string) string {
	//log.Println(contract)
	switch contract {
	case QUARTER_CONTRACT:
		return "CQ"
	case NEXT_WEEK_CONTRACT:
		return "NW"
	case THIS_WEEK_CONTRACT:
		return "CW"
	}
	return ""
}

func (ws *FuturesWs) adaptTime(tm string) int64 {
	format := "2006-01-02 15:04:05"
	day := time.Now().Format("2006-01-02")
	local, _ := time.LoadLocation("Asia/Chongqing")
	t, _ := time.ParseInLocation(format, day+" "+tm, local)
	return t.UnixNano() / 1e6

}
