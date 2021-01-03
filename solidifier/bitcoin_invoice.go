package main

import (
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	lightning "github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
	"github.com/lightningnetwork/lnd/zpay32"
)

func bitcoin_invoice(p *plugin.Plugin, params plugin.Params) (interface{}, int, error) {
	// make a normal invoice
	normalParams := map[string]interface{}{
		"msatoshi":    params.Get("msatoshi").String(),
		"label":       params.Get("label").String(),
		"description": params.Get("description").String(),
	}
	if expiry, ok := params["expiry"]; ok {
		normalParams["expiry"] = expiry
	}
	if preimage, ok := params["preimage"]; ok {
		normalParams["preimage"] = preimage
	}

	resp, err := p.Client.Call("invoice", normalParams)
	if err != nil {
		if errC, ok := err.(lightning.ErrorCommand); ok {
			return nil, errC.Code, errC
		}
		return nil, -1, err
	}

	// use data from that invoice to make a shadowed bitcoin invoice
	invoice, err := zpay32.Decode(resp.Get("bolt11").String(), &chaincfg.Params{
		Bech32HRPSegwit: "ex",
	})
	if err != nil {
		return nil, 801, err
	}

	invoice.Net = &chaincfg.RegressionNetParams // &chaincfg.MainNetParams 
	invoice.Destination = nil
	invoice.RouteHints = [][]zpay32.HopHint{
		{
			{
				NodeID:                    bridgeKey,
				ChannelID:                 getOurChannelWithBridge(p),
				FeeBaseMSat:               0,
				FeeProportionalMillionths: 0,
				CLTVExpiryDelta:           24,
			},
		},
	}

	privateKey, err := p.Client.GetPrivateKey()
	if err != nil {
		return nil, 802, err
	}

	bolt11, err := invoice.Encode(zpay32.MessageSigner{
		SignCompact: func(hash []byte) ([]byte, error) {
			return btcec.SignCompact(btcec.S256(), privateKey, hash, true)
		},
	})
	if err != nil {
		return nil, 803, err
	}

	return map[string]interface{}{
		"payment_hash": resp.Get("payment_hash").String(),
		"expires_at":   resp.Get("expires_at").String(),
		"bolt11":       bolt11,
	}, 0, nil
}
