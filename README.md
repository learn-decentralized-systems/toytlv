#   ToyTLV

[TLV][t] or Type-Length-Value is the default way to implement binary protocols.
You may ask, why not Protobuf? The answer is, Protobuf is a TLV protocol, as
almost every other binary protocol out there.

This lib implements a really simple TLV where every record type is a letter
(A-Z), while the length is either 8- or 32-bit little-endian integer. For
smaller records we use 8-bit, for records longer than 0xff bytes we use 32-bit.
That is an optimization to handle lots of tiny records. The type letter case
flags 32 or 8 ('A' for 32, 'a' for 8 bit).

The body of a record has arbitrary structure, ToyTLV mandates nothing in this
regard. Hint: nesting records is trivial. Sending JSON or Protobuf records is
equally so. The lib takes care of connecting, listening, reconnecting with 
exponential backoff, and otherwise managing the connections.

That is all it does.

[t]: https://en.wikipedia.org/wiki/Type%E2%80%93length%E2%80%93value
