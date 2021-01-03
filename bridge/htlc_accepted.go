package main

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	lightning "github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
	"github.com/tidwall/gjson"
)

var continueHTLC = map[string]interface{}{"result": "continue"}

func htlc_accepted(p *plugin.Plugin, params plugin.Params) (resp interface{}) {
	scid := params.Get("onion.short_channel_id").String()
	if scid == "0x0x0" {
		// payment coming to this node, accept it
		return continueHTLC
	}

	var bridgedChannel *gjson.Result
	var bridgedPeer *gjson.Result
	// scan bridged channels to see if this payment is intended to one of them
	bridgePeers, err := that.Call("listpeers")
	if err != nil {
		p.Logf("couldn't list peers on bridge, this is an error: %s", err)
		return continueHTLC
	}
	for _, peer := range bridgePeers.Get("peers").Array() {
		for _, channel := range peer.Get("channels").Array() {
			if channel.Get("short_channel_id").String() == scid {
				bridgedChannel = &channel
				bridgedPeer = &peer
			}
			if bridgedChannel != nil {
				break
			}
		}
	}
	if bridgedChannel == nil {
		// next channel is not on the bridge
		return continueHTLC
	}

	amount := params.Get("onion.forward_amount").String()
	hash := params.Get("htlc.payment_hash").String()
	p.Logf("bridging HTLC. amount=%s short_channel_id=%s hash=%s", amount, scid, hash)

	// here is the onion we must forward and to where
	nextonion := params.Get("onion.next_onion").String()
	firstHop := struct {
		ID         string `json:"id"`
		Direction  int64  `json:"direction"`
		AmountMsat string `json:"amount_msat"`
		Delay      int64  `json:"delay"`
	}{
		bridgedPeer.Get("id").String(),
		bridgedChannel.Get("direction").Int(),
		amount,
		params.Get("onion.outgoing_cltv_value").Int(),
	}

	if _, exists := paymentNotifications[hash]; exists {
		// payment already in course here
		return map[string]interface{}{
			"result": "reject",
		}
	}

	mutex.Lock()
	paymentNotifications[hash] = make(chan string)
	errorNotifications[hash] = make(chan string)
	mutex.Unlock()
	justContinue := make(chan bool)

	go func() {
		// this gives us time to listen for notifications down there
		time.Sleep(1 * time.Second)

		_, err = that.Call("sendonion", nextonion, firstHop, hash)
		if err != nil {
			p.Logf("error bridging! %s", err.Error())
			switch e := err.(type) {
			case lightning.ErrorTimeout:
				// do nothing, we will wait for a notification and handle this there
			case lightning.ErrorCommand:
				if e.Code <= 201 || e.Code == 204 {
					// local errors that happen before an actual pay attempt
					justContinue <- true
				}
				// otherwise do nothing
			default:
				// the command has failed somehow
				p.Logf("unexpected sendonion call failure: %s", err.Error())
				panic("something is very wrong!")
			}
			return
		}
	}()

	defer func() {
		mutex.Lock()
		delete(paymentNotifications, hash)
		delete(errorNotifications, hash)
		mutex.Unlock()
	}()

	// here we should get a notification from the other node
	select {
	case preimage := <-paymentNotifications[hash]:
		// ensure our preimage is correct
		preimageBytes, _ := hex.DecodeString(preimage)
		derivedHash := sha256.Sum256(preimageBytes)
		derivedHashHex := hex.EncodeToString(derivedHash[:])
		if derivedHashHex != hash {
			p.Logf("we have a preimage %s, but its hash %s didn't match the expected hash %s ", preimage, derivedHashHex, hash)
			panic("call the police, we're being robbed!")
		}

		p.Logf("bridge success. we have a preimage: %s - resolve", preimage)
		return map[string]interface{}{
			"result":      "resolve",
			"payment_key": preimage,
		}
	case failureOnion := <-errorNotifications[hash]:
		p.Logf("bridge attempt failed. we have an onion: %s", failureOnion)
		return map[string]interface{}{
			"result":        "fail",
			"failure_onion": failureOnion,
		}
	case <-justContinue:
		return continueHTLC
	}
}
