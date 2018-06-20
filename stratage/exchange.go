package stratage

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/morya/looptrader/conf"
	"github.com/morya/looptrader/log"
	"github.com/morya/looptrader/exproxy"
	"github.com/morya/looptrader/exproxy/model"
)

//
type Stratage struct {
	Symbol        string
	BaseCurrency  string
	QuoteCurrency string
	//	SellNumber    float64
	ExpectValue float64
	Balance     map[string]*model.BalanceContext
	Quote       *model.Quote

	exproxy *exproxy.EXProxy

	config *model.Configuration
	sync.RWMutex

	shuadanChan chan int
	accountChan chan int
}

//
func NewStratage(cfg *model.Configuration) (*Stratage, error) {
    proxy, err := exproxy.NewEXProxy(cfg.AppKey, cfg.AppSecret)
    if err != nil {
        return nil, err
    }

	list, err := proxy.GetCurrencies()
	if err != nil {
		return nil, err
	}

	if list.Status != 0 {
		return nil, fmt.Errorf("get currencies but return status is not 0")
	}

	var (
		base  string
		quote string
	)
	for _, v := range list.Data {
		if strings.HasPrefix(cfg.Symbol, v) {
			base = v
			quote = cfg.Symbol[len(v):]
			break
		}
	}

	if !strings.Contains(strings.Join(list.Data, ","), quote) {
		return nil, fmt.Errorf("symbol %s does not support", cfg.Symbol)
	}

	return &Stratage{
		Symbol:        cfg.Symbol,
		BaseCurrency:  base,
		QuoteCurrency: quote,
		exproxy:      proxy,
		config:        cfg,
		Balance:       make(map[string]*model.BalanceContext),
		accountChan:   make(chan int, 1),
		shuadanChan:   make(chan int, 1),
	}, nil
}

func (p *Stratage) AutoUpdate() {
	//go p.AutoUpdateTicker()
	//go p.AutoUpdateBalance()
	if p.config.AutoCheckOrder {
		go p.AutoCheckOrders()
	}
	time.Sleep(time.Second)
	go p.AutoShuaDan()
}

// 自动更新行情信息
func (p *Stratage) AutoUpdateTicker() {
	log.Logger.Infof("start auto update ticker task")
	var (
		interval int64 = conf.GetConfiguration().UpdateTickerInterval
		ticker   *model.Ticker
		quote    *model.Quote
		err      error
	)

	if interval < 500 {
		log.Logger.Infof("update ticker interval less than 500, set it to 500")
		interval = 500
	}

	tk := time.NewTicker(time.Duration(interval) * time.Millisecond)

	for {
		ticker, err = p.fcclient.GetTicker(p.Symbol)
		if err != nil {
			log.Logger.Errorf("get %s ticker failed. %s\n", p.Symbol, err)
		} else {
			if ticker.Status != 0 {
				log.Logger.Errorf("get ticker but return status is %d", ticker.Status)
			} else {
				quote, err = fcoin.ParseTicker(ticker)
				if err != nil {
					log.Logger.Errorf("%s\n", err)
				} else {
					p.Quote = quote
				}

			}
		}
		<-tk.C
	}
}

/*
定时检查订单是否很久未成交、部分成交
过期自动取消
*/
func (p *Stratage) AutoCheckOrders() {
	log.Logger.Infof("start auto check order task")
	var (
		interval int64 = conf.GetConfiguration().CheckOrderInterval
		orders   *model.OrderList
		err      error
		querys   map[string]string = map[string]string{
			"symbol": p.Symbol,
			"limit":  "10",
		}
		// 定时取消未成交、部分成交的订单
		states     []string = []string{"submitted", "partial_filled"}
		serverTime *model.ServerTime
		timeDValue int64
	)

	if interval < 500 {
		log.Logger.Infof("check order interval less than 500, set it to 500")
		interval = 500
	}

	tk := time.NewTicker(time.Duration(interval) * time.Millisecond)

	for {
		for _, state := range states {
			querys["states"] = state
			orders, err = p.fcclient.ListOrders(querys)
			if err != nil {
				log.Logger.Errorf("get orders failed. %s", err)
				continue
			}

			if orders.Status != 0 {
				log.Logger.Errorf("get orderlist but return status is %d", orders.Status)
				continue
			}

			serverTime, err = p.fcclient.GetServerTime()
			if err != nil {
				log.Logger.Errorf("get server time failed. %v", serverTime)
				serverTime = new(model.ServerTime)
				serverTime.Data = time.Now().UnixNano() / 1000000
			}
			for _, order := range orders.Data {
				log.Logger.Infof("order id %s, created at: %d", order.Id, order.CreatedAt)
				timeDValue = order.CreatedAt - serverTime.Data
				log.Logger.Infof("time d_value: %d", timeDValue)
				if math.Abs(float64(timeDValue)) > float64(p.config.RevokeOrderTime) {
					// invoke order
					log.Logger.Infof("cancel order id %s", order.Id)
					var corder *model.CancelOrder
					corder, err = p.fcclient.CancelOrder(order.Id)
					if err != nil {
						log.Logger.Infof("cancel order failed. %s", err)
					}
					if corder.Status != 0 {
						log.Logger.Infof("cancel order failed. %v", corder)
					}
				}
				time.Sleep(time.Second)
			}
		}
		<-tk.C
	}
}

func (p *Stratage) GetQuote() *model.Quote {
	p.RLock()
	defer p.RUnlock()
	return p.Quote
}

func (p *Stratage) Buy(price, amount string) (*model.Order, error) {
	return p.fcclient.CreateOrder(p.Symbol, "buy", "limit", price, amount)
}

func (p *Stratage) Sell(price, amount string) (*model.Order, error) {
	return p.fcclient.CreateOrder(p.Symbol, "sell", "limit", price, amount)
}

func (p *Stratage) GetAccountBalance() (*model.AccountBalance, error) {
	return p.fcclient.GetBalance()
}

func (p *Stratage) CancelOrders() {
	var (
		orders *model.OrderList
		err    error
		querys map[string]string = map[string]string{
			"symbol": p.Symbol,
			"limit":  "10",
		}
		states     []string = []string{"submitted", "partial_filled"}
		serverTime *model.ServerTime
		timeDValue int64
	)

	for _, state := range states {
		querys["states"] = state
		orders, err = p.fcclient.ListOrders(querys)
		if err != nil {
			log.Logger.Errorf("get orders failed. %s\n", err)
		} else {
			if orders.Status != 0 {
				log.Logger.Errorf("get orderlist but return status is %d", orders.Status)
			} else {
				serverTime, err = p.fcclient.GetServerTime()
				if err != nil {
					log.Logger.Errorf("get server time failed. %v", serverTime)
					serverTime = new(model.ServerTime)
					serverTime.Data = time.Now().UnixNano() / 1000000
				}
				for _, order := range orders.Data {
					log.Logger.Infof("order id %s, created at: %d", order.Id, order.CreatedAt)
					timeDValue = order.CreatedAt - serverTime.Data
					log.Logger.Infof("time d_value: %d", timeDValue)
					if math.Abs(float64(timeDValue)) > float64(p.config.RevokeOrderTime) {
						// invoke order
						log.Logger.Infof("cancel order id %s", order.Id)
						var corder *model.CancelOrder
						corder, err = p.fcclient.CancelOrder(order.Id)
						if err != nil {
							log.Logger.Infof("cancel order failed. %s", err)
						} else if corder.Status != 0 {
							log.Logger.Infof("cancel order failed. %v", corder)
						}
					}
					time.Sleep(time.Second)
				}
			}
		}
	}
}

func (p *Stratage) GetCurrentQuote() (*model.Quote, error) {
	ticker, err := p.fcclient.GetTicker(p.Symbol)
	if err != nil {
		return nil, err
	}

	if ticker.Status != 0 {
		return nil, err
	}

	return fcoin.ParseTicker(ticker)
}
