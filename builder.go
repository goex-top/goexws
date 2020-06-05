package goexws

import (
	"github.com/goex-top/goexws/binance"
	"github.com/goex-top/goexws/huobi"
	"github.com/goex-top/goexws/okex"
)

func SpotBuild(ex string) SpotWsApi {
	switch ex {
	case Spot_Binance:
		return binance.NewSpotWs()
	case Spot_Huobi:
		return huobi.NewSpotWs()
	case Spot_OKEx:
		return okex.NewSpotWs()
	default:
		return nil
	}
}

func FuturesBuild(ex string) FuturesWsApi {
	switch ex {
	case Futures_Binance:
		return nil
	case Futures_Huobi:
		return huobi.NewFutureWs()
	case Futures_OKEx:
		return okex.NewFuturesWs()
	default:
		return nil
	}
}
