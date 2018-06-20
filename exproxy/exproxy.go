package exproxy

import (
	"fmt"
	"sync"

	"github.com/morya/looptrader/exproxy/model"
)

type FuncExchangeBuilder func(key, sec string) IExchange

var _lock = new(sync.Mutex)
var register = make(map[string]FuncExchangeBuilder)

func RegisterExchange(name string, builder FuncExchangeBuilder) {
	_lock.Lock()
	defer _lock.Unlock()
	register[name] = builder
}

type IExchange interface {
	GetServerTime() (*model.ServerTime, error)
	GetCurrencies() (*model.Currencies, error)
}

// EXProxy exchange proxy
type EXProxy struct {
	key, secret string
	exchange    IExchange
}

func buildExchange(key, secret, name string) IExchange {
	_lock.Lock()
	defer _lock.Unlock()

	if builder, ok := register[name]; ok {
		return builder(key, secret)
	}
	return nil
}

func NewEXProxy(key, secret, name string) (*EXProxy, error) {
	var exchange = buildExchange(key, secret, name)
	if exchange == nil {
		return nil, fmt.Errorf("exchange not supported")
	}

	var p = &EXProxy{
		key:      key,
		secret:   secret,
		exchange: exchange,
	}

	return p, nil
}

func (this *EXProxy) GetServerTime() (*model.ServerTime, error) {
	return this.exchange.GetServerTime()
}

func (this *EXProxy) GetCurrencies() (*model.Currencies, error) {
	return this.exchange.GetCurrencies()
}
