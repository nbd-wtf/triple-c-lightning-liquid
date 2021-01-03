package main

import (
	"strings"

	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
)

func rpc_command(p *plugin.Plugin, params plugin.Params) (resp interface{}) {
	if p.Args.Get("tcll-hijack-commands").Bool() {
		switch params.Get("rpc_command.method").String() {
		case "pay":
			bolt11 := params.Get("rpc_command.params.0").String()
			if bolt11 == "" {
				bolt11 = params.Get("rpc_command.params.bolt11").String()
			}
			if strings.HasPrefix(bolt11, "lnbc") {
				resp, _, err := bitcoin_pay(p, params)
				if err == nil {
					return map[string]interface{}{
						"return": map[string]interface{}{
							"result": resp,
						},
					}
				} else {
					return map[string]interface{}{
						"return": map[string]interface{}{
							"error": err,
						},
					}
				}
			}
		case "invoice":
			resp, _, err := bitcoin_invoice(p, params)
			if err == nil {
				return map[string]interface{}{
					"return": map[string]interface{}{
						"result": resp,
					},
				}
			} else {
				return map[string]interface{}{
					"return": map[string]interface{}{
						"error": err,
					},
				}
			}
		}
	}

	return map[string]string{"result": "continue"}
}
