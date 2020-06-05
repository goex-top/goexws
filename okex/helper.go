package okex

import "time"

//
import (
	. "github.com/nntaoli-project/goex"
)

type tickerResponse struct {
	InstrumentId string  `json:"instrument_id"`
	Last         float64 `json:"last,string"`
	High24h      float64 `json:"high_24h,string"`
	Low24h       float64 `json:"low_24h,string"`
	BestBid      float64 `json:"best_bid,string"`
	BestAsk      float64 `json:"best_ask,string"`
	Volume24h    float64 `json:"volume_24h,string"`
	Timestamp    string  `json:"timestamp"`
}

type spotTickerResponse struct {
	InstrumentId  string  `json:"instrument_id"`
	Last          float64 `json:"last,string"`
	High24h       float64 `json:"high_24h,string"`
	Low24h        float64 `json:"low_24h,string"`
	BestBid       float64 `json:"best_bid,string"`
	BestAsk       float64 `json:"best_ask,string"`
	BaseVolume24h float64 `json:"base_volume_24h,string"`
	Timestamp     string  `json:"timestamp"`
}

type depthResponse struct {
	Bids         [][4]interface{} `json:"bids"`
	Asks         [][4]interface{} `json:"asks"`
	InstrumentId string           `json:"instrument_id"`
	Timestamp    string           `json:"timestamp"`
}

func adaptKLinePeriod(period int) int {
	granularity := -1
	switch period {
	case KLINE_PERIOD_1MIN:
		granularity = 60
	case KLINE_PERIOD_3MIN:
		granularity = 180
	case KLINE_PERIOD_5MIN:
		granularity = 300
	case KLINE_PERIOD_15MIN:
		granularity = 900
	case KLINE_PERIOD_30MIN:
		granularity = 1800
	case KLINE_PERIOD_1H, KLINE_PERIOD_60MIN:
		granularity = 3600
	case KLINE_PERIOD_2H:
		granularity = 7200
	case KLINE_PERIOD_4H:
		granularity = 14400
	case KLINE_PERIOD_6H:
		granularity = 21600
	case KLINE_PERIOD_1DAY:
		granularity = 86400
	case KLINE_PERIOD_1WEEK:
		granularity = 604800
	}
	return granularity
}

func adaptSecondsToKlinePeriod(seconds int) int {
	var p KlinePeriod
	switch seconds {
	case 60:
		p = KLINE_PERIOD_1MIN
	case 180:
		p = KLINE_PERIOD_3MIN
	case 300:
		p = KLINE_PERIOD_5MIN
	case 900:
		p = KLINE_PERIOD_15MIN
	case 1800:
		p = KLINE_PERIOD_30MIN
	case 3600:
		p = KLINE_PERIOD_1H
	case 7200:
		p = KLINE_PERIOD_2H
	case 14400:
		p = KLINE_PERIOD_4H
	case 21600:
		p = KLINE_PERIOD_6H
	case 86400:
		p = KLINE_PERIOD_1DAY
	case 604800:
		p = KLINE_PERIOD_1WEEK
	}
	return int(p)
}

func timeStringToInt64(t string) (int64, error) {
	timestamp, err := time.Parse(time.RFC3339, t)
	if err != nil {
		return 0, err
	}
	return timestamp.UnixNano() / int64(time.Millisecond), nil
}
