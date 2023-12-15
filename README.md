#   ToyTLV

[TLV][t] or Type-Length-Value is the default way to implement binary protocols.
You may ask, why not Protobuf? The answer is, Protobuf is TLV as virtually
every other binary protocol.

This lib implements a really simple TLV where every record type is a letter
(A-Z), while the length is either 8- or 32-bit little-endian integer. For
smaller records we use 8-bit, for records longer than 0xff bytes we use 32-bit.
The type letter is either uppercase or lowercase to flag 32 vs 8.

The body of a record has arbitrary structure, ToyTLV mandates nothing in this
regard. The lib can connect, listen, reconnect automatically (with exponential
backoff), and otherwise manages the connections. That is all it does.

[t]: https://en.wikipedia.org/wiki/Type%E2%80%93length%E2%80%93value
