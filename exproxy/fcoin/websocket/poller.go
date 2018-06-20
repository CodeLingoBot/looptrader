package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	//"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/morya/ex/alarm"
	"github.com/morya/ex/dumper"
	"github.com/morya/ex/sign"
	"github.com/morya/utils/log"
)

type SymbolMap map[string]string

type Poller struct {
	sync.Mutex
	waiter    *sync.WaitGroup
	_hbWaiter *sync.WaitGroup

	shouldStop  bool
	key, secret string

	Address  string
	conn     *websocket.Conn
	recvChan chan interface{}
	sendChan chan interface{}

	tickersMap SymbolMap // 行情记录表
	dealsMap   SymbolMap // 全所交易记录
}

func NewWebSockConn(addr string) (*websocket.Conn, error) {
	u, err := url.Parse(addr)
	log.Infof("connecting to %s", u.String())
	if err != nil {
		return nil, err
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Info(err)
		return nil, err
	}

	log.Info("connect ok")

	return conn, nil
}

func NewPoller(address string, key, secret string, onMsgChan chan interface{}) (*Poller, error) {
	sendChan := make(chan interface{}, 10)

	return &Poller{

		waiter:    new(sync.WaitGroup),
		_hbWaiter: new(sync.WaitGroup),
		key:       key,
		secret:    secret,

		recvChan: onMsgChan,
		sendChan: sendChan,

		tickersMap: make(SymbolMap),
		dealsMap:   make(SymbolMap),
	}, nil
}

func (p *Poller) SubDeals(symbol string) {

	p.Lock()
	defer p.Unlock()
	p.dealsMap[symbol] = symbol

	msg := newEventSubDeals(symbol)
	log.Debugf("SubDeals, msg %s", dumper.DumpObject(msg))
	p.conn.WriteJSON(msg)
}

func (p *Poller) SubTicker(symbol string) {
	p.Lock()
	defer p.Unlock()
	p.tickersMap[symbol] = symbol
	msg := newEventSubTicker(symbol)
	log.Debugf("SubTicker, msg %s", dumper.DumpObject(msg))
	p.conn.WriteJSON(msg)
}

func (p *Poller) UpdateUserInfo() error {
	log.Debug("send sub user info msg")

	if err := p.checkKeyAndSecret(); err != nil {
		return err
	}

	params := map[string]string{}
	p.AddSignToParams(p.key, p.secret, params)

	var msg = map[string]interface{}{
		"event":      "addChannel",
		"channel":    "ok_spot_userinfo",
		"parameters": params,
	}

	p.sendChan <- msg
	return nil
}

func (p *Poller) AddSignToParams(key, secret string, params map[string]string) error {
	params["api_key"] = key

	postForm := &url.Values{}
	for k, v := range params {
		postForm.Set(k, v)
	}

	// Encode() will sort and escape all params
	// and `secret_key` should be added at last
	payload := postForm.Encode()
	payload = payload + "&secret_key=" + secret
	payload2, _ := url.QueryUnescape(payload) // can't escape for sign

	sign, _ := sign.GetParamMD5Sign(secret, payload2)
	params["sign"] = sign
	return nil
}

func (p *Poller) checkKeyAndSecret() error {
	if len(p.key) == 0 || len(p.secret) == 0 {
		err := fmt.Errorf("key or secret is empty")
		return err
	}
	return nil
}

func (p *Poller) Login() error {
	if err := p.checkKeyAndSecret(); err != nil {
		return err
	}
	params := map[string]string{}
	p.AddSignToParams(p.key, p.secret, params)

	var msg = map[string]interface{}{
		"event":      "login",
		"parameters": params,
	}

	p.sendChan <- msg
	return nil
}

func (p *Poller) onRawMsg(ctx context.Context, cancel context.CancelFunc, bindata []byte) error {
	var raw interface{}

	var msgs = make([]*TagMsgCommReply, 0)
	err := json.Unmarshal(bindata, &msgs)
	if err != nil {
		return err
	}

	// fuck okex
	// websokcet api就是返回数组里面的元素
	// 估计是为了方便javascript客户端开发
	msg := msgs[0]

	if strings.HasPrefix(msg.Channel, "ok_sub_spot") && strings.HasSuffix(msg.Channel, "_ticker") {
		var t = make([]*TagMsgTickerAll, 0)
		err := json.Unmarshal(bindata, &t)
		if err != nil {
			return err
		}
		raw = t[0]

	} else if msg.Channel == "ok_spot_userinfo" {
		var uinfo = make([]*TagMsgUserInfo, 0)
		err := json.Unmarshal(bindata, &uinfo)
		if err != nil {
			return err
		}
		raw = uinfo[0]

	} else if strings.HasSuffix(msg.Channel, "_deals") {
		// ignore this msg
		var deals = make([]*TagMsgDeals, 0)
		err := json.Unmarshal(bindata, &deals)
		if err != nil {
			return err
		}
		raw = deals[0]

	} else if strings.HasSuffix(msg.Channel, "_balance") {
		var msglist = make([]*TagMsgBalanceUpdate, 0)
		err = json.Unmarshal(bindata, &msglist)
		if err == nil {
			raw = msglist[0]
		}

	} else if strings.HasPrefix(msg.Channel, "ok_sub_spot_") && strings.HasSuffix(msg.Channel, "_order") {
		var msglist = make([]*TagMsgOrderUpdate, 0)
		err = json.Unmarshal(bindata, &msglist)
		if err == nil {
			raw = msglist[0]
		}

	} else {
		var rawmsgs = make([]interface{}, 0)
		err = json.Unmarshal(bindata, &rawmsgs)
		if err == nil {
			log.Infof("[unknown] msg %s", dumper.DumpObject(rawmsgs[0]))
		}
	}

	if raw != nil {
		p.recvChan <- raw
	}

	return nil
}

func (p *Poller) SendLoop(ctx context.Context, cancel context.CancelFunc) {
	defer cancel()

	for {
		select {
		case msg := <-p.sendChan:
			switch msg.(type) {
			case int:
				log.Debugf("send heartbeat msg")
				now := time.Now()
				p.conn.WriteControl(websocket.PingMessage, []byte{}, now)

			default:
				err := p.conn.WriteJSON(msg)
				if err != nil {
					log.InfoError(err)
					p.conn.Close()
				}
			}

		case <-ctx.Done():
			log.Info("sendloop exit")
			return
		}
	}
}

func (p *Poller) RecvLoop(ctx context.Context, cancel context.CancelFunc) {
	defer p.conn.Close()
	defer cancel()

	var bindata []byte
	var err error

	for {
		_, bindata, err = p.conn.ReadMessage()
		if err != nil && !p.shouldStop {
			log.InfoError(err, "read error:")
			return
		}
		if p.shouldStop {
			return
		}

		log.Debugf("bindata %s", bindata)

		err = p.onRawMsg(ctx, cancel, bindata)
		if err != nil {
			log.InfoError(err)
		}
	}
}

func (p *Poller) Keeper(ctx context.Context, cancel context.CancelFunc) {
	err := p.Login()
	if err != nil {
		time.Sleep(time.Second)
		log.InfoError(err)
		cancel()
		return
	} else {
		log.Info("login ok")
	}
	go p.HeartBeat(ctx, cancel)
	go p.SendLoop(ctx, cancel)

	go func() {
		// 首次连接尽快更新用户状态
		time.Sleep(time.Second)
		p.UpdateUserInfo()
	}()

	p.RecvLoop(ctx, cancel)
}

func (p *Poller) Connect() error {
	conn, err := NewWebSockConn(p.Address)
	if err != nil {
		return err
	}
	p.conn = conn
	return nil
}

func (p *Poller) HeartBeat(ctx context.Context, cancel context.CancelFunc) {

	p._hbWaiter.Add(1)
	defer cancel()
	defer p._hbWaiter.Done()

	log.Debug("start heartbeat")

	hb := time.NewTicker(time.Second * 15)
	balance := time.NewTicker(time.Second * 60)

	for {
		select {
		case <-ctx.Done():
			log.Info("quit heartbeat")
			return

		case <-hb.C:
			log.Debug("poller send heartbeat")
			p.sendChan <- 1

		case <-balance.C:
			p.UpdateUserInfo()
		}
	}
}

func (p *Poller) ReSubscribe() {
	p.Lock()
	defer p.Unlock()
	var msg *TagMsgEventSub

	for symbol, _ := range p.tickersMap {
		msg = newEventSubTicker(symbol)
		p.sendChan <- msg
	}

	for symbol, _ := range p.dealsMap {
		msg = newEventSubDeals(symbol)
		p.sendChan <- msg
	}
}

func (p *Poller) Stop() {
	p.shouldStop = true
	p.conn.Close()
	p._hbWaiter.Wait()
	p.waiter.Wait()
}

func (p *Poller) Run() {
	var ctx context.Context
	var cancel context.CancelFunc

	var connectFail int

	p.waiter.Add(1)

	defer func() {
		if cancel != nil {
			cancel()
		}
		p.waiter.Done()
	}()

	for {
		if p.shouldStop {
			log.Info("poller exiting..")
			return
		}

		connectFail = 0

		ctx, cancel = context.WithCancel(context.Background())
		p.Keeper(ctx, cancel)

		if p.shouldStop {
			log.Info("poller exiting..")
			return
		}

		for {
			err := p.Connect()
			if err != nil {
				connectFail++
				if connectFail > 20 {
					alarm.FangtangNotify("websocket告警", "无法连接服务器")
					return
				}
				time.Sleep(time.Second * 5)
				log.Info("restart connection failed")
				continue
			} else {
				log.Info("restart connection succ")
				p.ReSubscribe()
				break
			}
		}
	}
}
