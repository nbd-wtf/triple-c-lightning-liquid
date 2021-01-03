package main

import (
	"sync"

	lightning "github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
)

var this *lightning.Client
var that *lightning.Client

var mutex = sync.Mutex{}
var paymentNotifications = make(map[string]chan string)
var errorNotifications = make(map[string]chan string)

func main() {
	p := plugin.Plugin{
		Name:    "tcll-bridge",
		Version: "v1.0",
		Dynamic: false,
		Options: []plugin.Option{
			{"tcll-other-rpc", "string", "", "lightning-rpc path of the other c-lightning node we're going to relay payments to"},
		},
		Hooks: []plugin.Hook{
			{
				"htlc_accepted",
				htlc_accepted,
			},
		},
		RPCMethods: []plugin.RPCMethod{
			{
				"bridge_sendpay_success",
				"payment_preimage payment_hash",
				"Accepts notifications from the other node which we're bridging payments with.",
				"",
				func(p *plugin.Plugin, params plugin.Params) (interface{}, int, error) {
					hash := params.Get("payment_hash").String()
					preimage := params.Get("payment_preimage").String()

					if preimageChan, ok := paymentNotifications[hash]; ok {
						preimageChan <- preimage
					}

					return nil, 0, nil
				},
			},
			{
				"bridge_sendpay_failure",
				"payment_hash onionreply",
				"Accepts notifications from the other node which we're bridging payments with.",
				"",
				func(p *plugin.Plugin, params plugin.Params) (interface{}, int, error) {
					hash := params.Get("payment_hash").String()
					onionreply := params.Get("onionreply").String()

					if onionReplyChan, ok := errorNotifications[hash]; ok {
						onionReplyChan <- onionreply
					}

					return nil, 0, nil
				},
			},
		},
		Subscriptions: []plugin.Subscription{
			{
				"sendpay_success",
				func(p *plugin.Plugin, params plugin.Params) {
					preimage := params.Get("sendpay_success.payment_preimage").String()
					hash := params.Get("sendpay_success.payment_hash").String()
					p.Logf("sending success preimage to bridge %s: %s", hash, preimage)
					that.Call("bridge_sendpay_success", preimage, hash)
				},
			},
			{
				"sendpay_failure",
				func(p *plugin.Plugin, params plugin.Params) {
					if params.Get("sendpay_failure.code").Int() == 202 {
						hash := params.Get("sendpay_failure.data.payment_hash").String()
						onion := params.Get("sendpay_failure.data.onionreply").String()
						p.Logf("sending failure onion to bridge %s: %s", hash, onion)
						that.Call("bridge_sendpay_failure", hash, onion)
					}
				},
			},
		},
		OnInit: func(p *plugin.Plugin) {
			this = p.Client

			otherRPC := p.Args["tcll-other-rpc"].(string)

			p.Logf("using RPC socket for the other network at: %s", otherRPC)
			that = &lightning.Client{
				Path: otherRPC,
			}
		},
	}

	p.Run()
}
