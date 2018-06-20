package websocket

import "fmt"

const (
	// -1:已撤销  0:未成交  1:部分成交  2:完全成交 3:撤单处理中
	OID_Trade_Canceled  int = -1
	OID_Trade_UnFinish  int = 0
	OID_Trade_Partial   int = 1
	OID_Trade_Finish    int = 2
	OID_Trade_Canceling int = 3
)

type TagBodyTicker struct {
	Buy       float64 `json:"buy,string"`
	Change    float64 `json:"change,string"`
	Close     float64 `json:"close,string"`
	DayHigh   float64 `json:"dayHigh,string"`
	DayLow    float64 `json:"dayLow,string"`
	High      float64 `json:"high,string"`
	Last      float64 `json:"last,string"`
	Low       float64 `json:"low,string"`
	Open      float64 `json:"open,string"`
	Sell      float64 `json:"sell,string"`
	Timestamp int64   `json:"timestamp"`
	Vol       float64 `json:"vol,string"`
}

type wsInnerFund struct {
	Usdt float64 `json:"usdt,string"`
	Btc  float64 `json:"btc,string"`
	Ltc  float64 `json:"ltc,string"`
	Eth  float64 `json:"eth,string"`
	Eos  float64 `json:"eos,string"`
	Etc  float64 `json:"etc,string"`
	Elf  float64 `json:"elf,string"`
	Bch  float64 `json:"bch,string"`
	// Vee  float64 `json:"vee,string"`
}

type TTagMapInnerFund map[string]string

type TagBodyUserInfo struct {
	Result bool `json:"result"`
	Info   struct {
		Funds struct {
			Free    TTagMapInnerFund `json:"free"`
			Freezed TTagMapInnerFund `json:"freezed"`
		} `json:"funds"`
	} `json:"info"`
}

// {"binary":0,"channel":"addChannel","data":{"result":true,"channel":"ok_sub_futureusd_userinfo"}}
type TagMsgCommReply struct {
	Binary  int    `json:"binary"`
	Channel string `json:"channel"`
}

type TagMsgEventSub struct {
	Event   string `json:"event"`
	Channel string `json:"channel"`
}

type TagMsgUserInfo struct {
	TagMsgCommReply
	Data TagBodyUserInfo `json:"data"`
}

type TagMsgTickerAll struct {
	TagMsgCommReply
	Data *TagBodyTicker `json:"data"`
}

/*
{
  "binary": 0,
  "channel": "ok_sub_spot_eos_usdt_balance",
  "data": {
    "info": {
      "free": {
        "usdt": 608.053747110309
      },
      "freezed": {
        "usdt": 1.245379
      }
    }
  }
}
*/
type TagMsgBalanceUpdate struct {
	TagMsgCommReply
	Data struct {
		Info struct {
			Free    map[string]float64 `json:"free"`
			Freezed map[string]float64 `json:"freezed"`
		} `json:"info"`
	} `json:"data"`
}

type TagMsgDeals struct {
	TagMsgCommReply
	Data [][]string
}

type TagMsgOrderUpdate struct {
	TagMsgCommReply
	Data struct {
		Symbol  string `json:'symbol'`
		OrderId int64
		Status  int `json:"status"` // -1:已撤销  0:未成交  1:部分成交  2:完全成交 3:撤单处理中

		TradeType            string  `json:"string"` // sell,buy
		TradeUnitPrice       float64 `json:"tradeUnitPrice,string"`
		TradePrice           float64 `json:"tradePrice,string"`
		CreatedDate          int64   `json:"createdDate,string"`
		UnTrade              float64 `json:"unTrade,string"`
		AveragePrice         float64 `json:"averagePrice,string"`
		TradeAmount          float64 `json:"tradeAmount,string"`
		CompletedTradeAmount float64 `json:"completedTradeAmount,string"`
	}
}

func newEventSubDeals(symbol string) *TagMsgEventSub {
	return &TagMsgEventSub{
		Event:   "addChannel",
		Channel: fmt.Sprintf("ok_sub_spot_%s_deals", symbol),
	}
}

func newEventSubTicker(symbol string) *TagMsgEventSub {
	return &TagMsgEventSub{
		Event:   "addChannel",
		Channel: fmt.Sprintf("ok_sub_spot_%s_ticker", symbol),
	}
}

func newEventSubUserInfo() *TagMsgEventSub {
	return &TagMsgEventSub{
		Event:   "addChannel",
		Channel: "ok_spot_userinfo",
	}
}
