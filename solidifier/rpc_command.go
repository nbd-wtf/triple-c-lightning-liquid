package main

import (
	"github.com/fiatjaf/lightningd-gjson-rpc/plugin"
)

func rpc_command(p *plugin.Plugin, params plugin.Params) (resp interface{}) {
	switch params.Get("rpc_command.method").String() {
	case "getroute":
		var err error
		iparams := params.Get("rpc_command.params").Value()
		if arrParams, ok := iparams.([]interface{}); ok {
			_, err = p.Client.Call("getroute", arrParams...)
		} else {
			_, err = p.Client.Call("getroute", iparams)
		}
		if err == nil {
			// getroute worked, so this is a liquid invoice
			// we won't do anything fancy
			return map[string]interface{}{"result": "continue"}
		}

		p.Logf("will ask a route to bitcoin %v", params.Get("rpc_command.params").Value())

		result, commandErr := getBitcoinRoute(p, params.Get("rpc_command.params"))

		p.Log(result)
		p.Log(commandErr)
		if commandErr != nil {
			return map[string]interface{}{
				"return": map[string]interface{}{
					"error": map[string]interface{}{
						"code":    commandErr.Code,
						"message": commandErr.Message,
					},
				},
			}
		}

		return map[string]interface{}{
			"return": map[string]interface{}{
				"result": result,
			},
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

	return map[string]string{"result": "continue"}
}
