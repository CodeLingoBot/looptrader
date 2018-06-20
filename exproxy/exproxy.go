package exproxy

import (
        "github.com/morya/looptrader/exproxy/fcoin/restful"
        "github.com/morya/looptrader/exproxy/fcoin/websocket"
)

type EXProxy struct {
    fcoinRest *restful.Client
    fcPoller *websocket.Poller
}

func NewEXProxy() (*EXProxy, error){
    return &EXProxy{}, nil
}

func (this*EXProxy)GetServerTime() (*model.ServerTime, error) {
    return fcoinRest.GetServerTime()
}

func (this*EXProxy) GetCurrencies() (*model.Currencies, error) {
    return fcoinRest.GetCurrencies()
}
