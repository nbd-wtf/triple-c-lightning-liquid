<a href="https://nbd.wtf"><img align="right" height="196" src="https://user-images.githubusercontent.com/1653275/194609043-0add674b-dd40-41ed-986c-ab4a2e053092.png" /></a>

to test this you'll need 4 lightningds running: 2 on bitcoin regtest, 2 on liquid.

A: regtest
B: regtest
C: liquid
D: liquid

1. copy `hsm_secret` from C into B so they have the same nodeid.
2. install `bridge` in both B and C.
3. install `solidifier` in D.
4. set `tcll-other-rpc` on B to the lightning-rpc path of C.
5. set `tcll-other-rpc` on C to the lightning-rpc path of B.
6. set `tcll-bridge-id` on D to the nodeid of C.
7. open a channel between A and B and another channel between C and D.

now you can use `bitcoin_invoice` to generate invoices on D that can be paid by A

and use `bitcoin_pay` on D to pay invoices generated by A.
