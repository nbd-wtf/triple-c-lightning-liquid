module github.com/fiatjaf/triple-c-lightning-liquid

go 1.15

require (
	github.com/brianolson/cbor_go v1.0.0
	github.com/btcsuite/btcd v0.20.1-beta.0.20200515232429-9f0179fd2c46
	github.com/fiatjaf/lightningd-gjson-rpc v1.1.0
	github.com/lightningnetwork/lnd v0.10.1-beta
	github.com/tidwall/gjson v1.6.0
)

replace github.com/fiatjaf/lightningd-gjson-rpc => /home/fiatjaf/comp/lightningd-gjson-rpc
