package main

import (
	"encoding/hex"
	"errors"
	"log"
	"time"

	cbor "github.com/brianolson/cbor_go"
	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
)

const GETROUTE_MESSAGE = "9aa1" // 39585 in hex
const ROUTEREPLY_MESSAGE = "9aa3"

var routeChan = make(chan map[string]interface{})

func bitcoin_pay(p *plugin.Plugin, params plugin.Params) (interface{}, int, error) {
	res, err := p.Client.Call("decodepay", params.Get("bolt11").String())
	if err != nil {
		return nil, -1, err
	}

	getRoute, _ := cbor.Dumps(map[string]interface{}{
		"id":         res.Get("payee").String(),
		"msatoshi":   res.Get("msatoshi").Int(),
		"riskfactor": 10,
	})

	payload := GETROUTE_MESSAGE + hex.EncodeToString(getRoute)
	p.Client.Call("dev-sendcustommsg", p.Args["tcll-bridge-id"].(string), payload)

	select {
	case route := <-routeChan:
		log.Print("got a route: ", route)

		// TODO increment this route with the liquid side

		// TODO send payment

	case <-time.After(time.Second * 3):
		// no route.
	}

	return nil, -1, errors.New("didn't get a route reply from bridge.")
}

func custommsg(p *plugin.Plugin, params plugin.Params) (resp interface{}) {
	message, _ := hex.DecodeString(params.Get("message").String())

	messageType := hex.EncodeToString(message[4:6])
	if messageType == ROUTEREPLY_MESSAGE {
		var route map[string]interface{}
		err := cbor.Loads(message[6:], &route)
		if err != nil {
			p.Logf("got invalid cbor on routereply")
		} else {
			routeChan <- route
		}
	}

	return map[string]interface{}{"result": "continue"}
}
