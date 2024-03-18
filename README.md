#   ToyTLV

[TLV][t] or Type-Length-Value is the default way to implement
binary protocols. You may ask, why not Protobuf? The answer is,
Protobuf is a TLV protocol, as almost every other binary
protocol out there. ToyTLV is a bare-bones TLV, nothing else.

This lib implements a really simple TLV where every record type
is a letter (A-Z), while the length is either 8- or 32-bit. The
body of a record has arbitrary structure, ToyTLV mandates
nothing in this regard. Hint: nesting TLV records is trivial.

More formally, a ToyTLV record can go in 3 forms:

 1. long: the type letter is uppercase [A-Z], the length is a
    little-endian uint32,
 2. short: the type letter is lowercase [a-z], the length is
    uint8. That is merely an optimization to handle lots of
    small records (like short strings).
 3. tiny: the type is known in advance, the length is ASCII
    [0-9]. This is for tiny records (e.g. small ints), where
    one byte of overhead can make a difference (that happens).

The lib implements basic ToyTLV file and network I/O. That is:
 - reading/writing TLV files,
 - basic TLV over TCP fun: connecting, listening, reconnecting
   with exponential backoff, and otherwise managing the
   connections (see TCPDepot).

That is all it does.

[t]: https://en.wikipedia.org/wiki/Type%E2%80%93length%E2%80%93value
