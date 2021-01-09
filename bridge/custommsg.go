package main

import (
	"encoding/hex"

	cbor "github.com/brianolson/cbor_go"
	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
)

const GETROUTE_MESSAGE = "9aa1" // 39585 in hex
const ROUTEREPLY_MESSAGE = "9aa3"

func custommsg(p *plugin.Plugin, params plugin.Params) (resp interface{}) {
	peer := params.Get("peer_id").String()
	message, _ := hex.DecodeString(params.Get("message").String())

	messageType := hex.EncodeToString(message[4:6])
	if messageType == GETROUTE_MESSAGE {
		var params map[string]interface{}
		err := cbor.Loads(message[6:], &params)
		if err != nil {
			p.Logf("got invalid cbor on getroute")
		} else {
			getRoute, err := that.Call("getroute", params)
			if err != nil {
				p.Logf("fail to getroute: %s", err)
			}

			route, _ := cbor.Dumps(getRoute.Get("route").Value())

			// add bridge fee TODO

			payload := ROUTEREPLY_MESSAGE + hex.EncodeToString(route)
			this.Call("dev-sendcustommsg", peer, payload)
		}
	}

	return map[string]interface{}{"result": "continue"}
}
