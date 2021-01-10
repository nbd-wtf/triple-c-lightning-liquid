package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	cbor "github.com/brianolson/cbor_go"
	lightning "github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
	"github.com/tidwall/gjson"
)

const GETROUTE_MESSAGE = "9aa1" // 39585 in hex
const ROUTEREPLY_MESSAGE = "9aa3"

var bitcoinHopsChan = make(chan gjson.Result)

func getBitcoinRoute(
	p *plugin.Plugin,
	params gjson.Result,
) (interface{}, *lightning.ErrorCommand) {
	inv, err := p.Client.Call("decodepay", params.Get("bolt11").String())
	if err != nil {
		return nil, &lightning.ErrorCommand{
			Code:    -1,
			Message: fmt.Sprintf("failed to decode invoice: %s", err.Error()),
		}
	}

	msatoshiToPay := inv.Get("msatoshi").Int()
	if msat := params.Get("msatoshi").Int(); msat > 0 {
		msatoshiToPay = msat
	}

	getRoute, _ := cbor.Dumps(map[string]interface{}{
		"id":         inv.Get("payee").String(),
		"msatoshi":   msatoshiToPay,
		"riskfactor": 10,
	})

	payload := GETROUTE_MESSAGE + hex.EncodeToString(getRoute)
	p.Client.Call("dev-sendcustommsg", p.Args["tcll-bridge-id"].(string), payload)

	select {
	case bitcoinHops := <-bitcoinHopsChan:
		// increment this route with the liquid side
		delayToAddOnLiquid := bitcoinHops.Get("0.delay").Int()
		msatNeededByBridge := bitcoinHops.Get("0.msatoshi").Int()
		bitcoinHopsLen := int(bitcoinHops.Get("#").Int())

		liquidRoute, err := p.Client.Call("getroute", map[string]interface{}{
			"id":         p.Args["tcll-bridge-id"].(string),
			"msatoshi":   msatNeededByBridge,
			"riskfactor": 10,
		})
		if err != nil {
			return nil, &lightning.ErrorCommand{
				Code:    -1,
				Message: "Couldn't find a route from us to bridge.",
			}
		}
		liquidHops := liquidRoute.Get("route")
		liquidHopsLen := int(liquidHops.Get("#").Int())

		// build a single hops array
		allHops := make([]interface{}, liquidHopsLen+bitcoinHopsLen)
		for i, hop := range liquidHops.Array() {
			// account for liquid blocktimes
			delay := (hop.Get("delay").Int() + int64(delayToAddOnLiquid)) * 10

			allHops[i] = map[string]interface{}{
				"id":        hop.Get("id").String(),
				"channel":   hop.Get("channel").String(),
				"direction": hop.Get("direction").Int(),
				"msatoshi":  hop.Get("msatoshi").Int(),
				"delay":     delay,
				"style":     hop.Get("style").String(),
			}
		}
		for i, hop := range bitcoinHops.Array() {
			allHops[i+liquidHopsLen] = map[string]interface{}{
				"id":        hop.Get("id").String(),
				"channel":   hop.Get("channel").String(),
				"direction": hop.Get("direction").Int(),
				"msatoshi":  hop.Get("msatoshi").Int(),
				"delay":     hop.Get("delay").Int(),
				"style":     hop.Get("style").String(),
			}
		}

		return map[string]interface{}{"route": allHops}, nil

	case <-time.After(time.Second * 3):
		// no route.
	}

	return nil, &lightning.ErrorCommand{
		Code:    -1,
		Message: "Bridge took too long to reply with a route.",
	}
}

func custommsg(p *plugin.Plugin, params plugin.Params) (resp interface{}) {
	message, _ := hex.DecodeString(params.Get("message").String())

	messageType := hex.EncodeToString(message[4:6])
	if messageType == ROUTEREPLY_MESSAGE {
		var hops []map[string]interface{}
		err := cbor.Loads(message[6:], &hops)
		if err != nil {
			p.Logf("got invalid cbor on routereply")
			return
		}

		hopsj, err := json.Marshal(hops)
		if err != nil {
			p.Log("got invalid route from bridge.")
			return
		}

		bitcoinHopsChan <- gjson.ParseBytes(hopsj)
	}

	return map[string]interface{}{"result": "continue"}
}
