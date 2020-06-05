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

type SpotWs struct {
	v3Ws           *baseWs
	tickerCallback func(*Ticker)
	depthCallback  func(*Depth)
	tradeCallback  func(*Trade)
	klineCallback  func(*Kline, int)
}

func NewSpotWs() *SpotWs {
	ws := &SpotWs{}
	ws.v3Ws = NewOKExV3Ws(ws.handle)
	return ws
}

func (ws *SpotWs) TickerCallback(tickerCallback func(*Ticker)) {
	ws.tickerCallback = tickerCallback
}

func (ws *SpotWs) DepthCallback(depthCallback func(*Depth)) {
	ws.depthCallback = depthCallback
}

func (ws *SpotWs) TradeCallback(tradeCallback func(*Trade)) {
	ws.tradeCallback = tradeCallback
}

func (ws *SpotWs) KlineCallback(call func(*Kline, int)) {
	ws.klineCallback = call
}

func (ws *SpotWs) KLineCallback(klineCallback func(kline *Kline, period int)) {
	ws.klineCallback = klineCallback
}

func (ws *SpotWs) SetCallbacks(tickerCallback func(*Ticker),
	depthCallback func(*Depth),
	tradeCallback func(*Trade),
	klineCallback func(*Kline, int)) {
	ws.tickerCallback = tickerCallback
	ws.depthCallback = depthCallback
	ws.tradeCallback = tradeCallback
	ws.klineCallback = klineCallback
}

func (ws *SpotWs) SubscribeDepth(currencyPair CurrencyPair, size int) error {
	if ws.depthCallback == nil {
		return errors.New("please set depth callback func")
	}

	return ws.v3Ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{fmt.Sprintf("spot/depth5:%s", currencyPair.ToSymbol("-"))}})
}

func (ws *SpotWs) SubscribeTicker(currencyPair CurrencyPair) error {
	if ws.tickerCallback == nil {
		return errors.New("please set ticker callback func")
	}
	return ws.v3Ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{fmt.Sprintf("spot/ticker:%s", currencyPair.ToSymbol("-"))}})
}

func (ws *SpotWs) SubscribeTrade(currencyPair CurrencyPair) error {
	if ws.tradeCallback == nil {
		return errors.New("please set trade callback func")
	}
	return ws.v3Ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{fmt.Sprintf("spot/trade:%s", currencyPair.ToSymbol("-"))}})
}

func (ws *SpotWs) SubscribeKline(currencyPair CurrencyPair, period int) error {
	if ws.klineCallback == nil {
		return errors.New("place set kline callback func")
	}

	seconds := adaptKLinePeriod(period)
	if seconds == -1 {
		return fmt.Errorf("unsupported kline period %d in okex", period)
	}

	return ws.v3Ws.Subscribe(map[string]interface{}{
		"op":   "subscribe",
		"args": []string{fmt.Sprintf("spot/candle%ds:%s", seconds, currencyPair.ToSymbol("-"))}})
}

func (ws *SpotWs) getCurrencyPair(instrumentId string) CurrencyPair {
	return NewCurrencyPair3(instrumentId, "-")
}

func (ws *SpotWs) handle(ch string, data json.RawMessage) error {
	var (
		err           error
		tickers       []spotTickerResponse
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
		candleResponse []struct {
			Candle       []string `json:"candle"`
			InstrumentId string   `json:"instrument_id"`
		}
	)

	switch ch {
	case "spot/ticker":
		err = json.Unmarshal(data, &tickers)
		if err != nil {
			return err
		}

		for _, t := range tickers {
			date, _ := time.Parse(time.RFC3339, t.Timestamp)
			ws.tickerCallback(&Ticker{
				Pair: ws.getCurrencyPair(t.InstrumentId),
				Last: t.Last,
				Buy:  t.BestBid,
				Sell: t.BestAsk,
				High: t.High24h,
				Low:  t.Low24h,
				Vol:  t.BaseVolume24h,
				Date: uint64(date.UnixNano() / int64(time.Millisecond)),
			})
		}
		return nil
	case "spot/depth5":
		err := json.Unmarshal(data, &depthResp)
		if err != nil {
			//logger.Error(err)
			return err
		}
		if len(depthResp) == 0 {
			return nil
		}

		dep.Pair = ws.getCurrencyPair(depthResp[0].InstrumentId)
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
	case "spot/trade":
		err := json.Unmarshal(data, &tradeResponse)
		if err != nil {
			//logger.Error("unmarshal error :", err)
			return err
		}

		for _, resp := range tradeResponse {
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
				Pair:   ws.getCurrencyPair(resp.InstrumentId),
			})
		}
		return nil
	default:
		if strings.HasPrefix(ch, "spot/candle") {
			err := json.Unmarshal(data, &candleResponse)
			if err != nil {
				return err
			}
			periodMs := strings.TrimPrefix(ch, "spot/candle")
			periodMs = strings.TrimSuffix(periodMs, "s")
			for _, k := range candleResponse {
				pair := ws.getCurrencyPair(k.InstrumentId)
				tm, _ := time.Parse(time.RFC3339, k.Candle[0])
				ws.klineCallback(&Kline{
					Pair:      pair,
					Timestamp: tm.Unix(),
					Open:      ToFloat64(k.Candle[1]),
					Close:     ToFloat64(k.Candle[4]),
					High:      ToFloat64(k.Candle[2]),
					Low:       ToFloat64(k.Candle[3]),
					Vol:       ToFloat64(k.Candle[5]),
				}, adaptSecondsToKlinePeriod(ToInt(periodMs)))
			}
			return nil
		}
	}

	return fmt.Errorf("unknown websocket message: %s", string(data))
}

func (ws *SpotWs) getKlinePeriodFormChannel(channel string) int {
	metas := strings.Split(channel, ":")
	if len(metas) != 2 {
		return 0
	}
	i, _ := strconv.ParseInt(metas[1], 10, 64)
	return int(i)
}
