package main

import (
	"encoding/hex"
	"os"

	"github.com/btcsuite/btcd/btcec"
	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
)

var bridgeKey *btcec.PublicKey
var bridgeChannel uint64

func main() {
	p := plugin.Plugin{
		Name:    "tcll-solidifier",
		Version: "v1.0",
		Dynamic: false,
		Options: []plugin.Option{
			{"tcll-bridge-id", "string", "", "Id of the bridge node (the bridge node is actually two different nodes, but using the same seed)"},
		},
		Hooks: []plugin.Hook{
			{
				"rpc_command",
				rpc_command,
			},
			{
				"custommsg",
				custommsg,
			},
		},
		OnInit: func(p *plugin.Plugin) {
			bridgeID := p.Args["tcll-bridge-id"].(string)

			bridgeIDBytes, err := hex.DecodeString(bridgeID)
			if err != nil {
				p.Logf("'tcll-bridge-id' is not a valid hex string.")
				os.Exit(1)
			}

			bridgeKey, err = btcec.ParsePubKey(bridgeIDBytes, btcec.S256())
			if err != nil {
				p.Logf("'tcll-bridge-id' is not a valid pubkey.")
				os.Exit(1)
			}

			_, err = p.Client.Call("connect", bridgeID)
			if err != nil {
				p.Logf("couldn't connect to 'tcll-bridge-id' as of now. make sure the bridge is online otherwise payments will fail.")
			}

			scid := getOurChannelWithBridge(p)
			if scid == 0 {
				p.Logf("we don't have a channel with the bridge node. be sure to open one or payments will fail.")
			}
		},
	}

	p.Run()
}
