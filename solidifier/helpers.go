package main

import (
	"strconv"
	"strings"

	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
)

func getOurChannelWithBridge(p *plugin.Plugin) uint64 {
	bridgeID := p.Args["tcll-bridge-id"].(string)

	peers, err := p.Client.Call("listpeers", bridgeID)
	if err != nil {
		p.Logf("couldn't listpeers to check our connection with the bridge.")
		return 0
	}

	scid := peers.Get("peers.0.channels.0.short_channel_id").String()
	if scid == "" {
		p.Logf("we don't have a channel with the bridge!")
		return 0
	}

	decoded, _ := decodeShortChannelId(scid)
	return decoded
}

func decodeShortChannelId(scid string) (uint64, error) {
	spl := strings.Split(scid, "x")

	x, err := strconv.ParseUint(spl[0], 10, 64)
	if err != nil {
		return 0, err
	}
	y, err := strconv.ParseUint(spl[1], 10, 64)
	if err != nil {
		return 0, err
	}
	z, err := strconv.ParseUint(spl[2], 10, 64)
	if err != nil {
		return 0, err
	}

	return ((x & 0xFFFFFF) << 40) | ((y & 0xFFFFFF) << 16) | (z & 0xFFFF), nil
}
