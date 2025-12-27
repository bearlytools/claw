# Claw

Claw's provides an easy alternative to Protocol Buffers that is free, performant and easy to use.
The format is open and we currently supply code generation for Go.

The whys for using it:

* Language generation is simple and easy to use, unlike protoc
* Lazy decode, which makes processing when using a subset of fields faster
* With no changes to any fields, encoding a received message does not requires re-encoding
* Provides the ability to send patches for changes and apply them instead of sending entire messages
* RPC system that can use TCP, HTTP 1.1/2 and Unix Domain Sockets
* You can decode messages without the IDL (which only means no field names)
* For Go, supports reflection 
* Can detect fields being set vs just having the zero value via an option 

The why nots:

* Proto has better language support
* Proto is supported by Google (though they break it all the time)
* Claw is likely bigger on the wire on the average case due to fixed sizes and padding vs variable length encoding

### Goals:

* Be friendly to the machine at the cost of wire size
* Zero heap allocations for data conversion to scaler types (for Go, Rust and C++)
* Lazy init, which has proven to be substantial faster
* Close to zero heap allocations for decode
* One pass encode/decode
* Supportable in Go, Rust, Zig, Javascript
* Import statements that make sense instead of that protoc nonsense
* Allow any type of serialization on top of our binary serialization if the other format can support our types
* The binary format should be fairly easy to understand if you understand binary file formats
* Detection of zero value vs not set

### Non-goals:
* Human readable output
* Small wire size (we want speed for the cost of size)

The wire size choice is probably the one to be the most contentious. It is my belief that variable length encoding buys little in terms of wire savings. I *believe* that even simple compression like zippy will eliminate any file size issue and will be substantially faster than variable length encoding (especially since you are likely to do both for protocol buffers).  In addition, with something like gzip, I imagine the numbers will almost be indistiguishable. 

But overall this format will be more in size. All entries are a minimum for 8 bytes (64 bits). In some things like lists of bools, we bit pack with great efficiency.  But other things like 64 bit numbers, we use 16 bytes. Lists of numbers are more efficient. This pushes our size closer to JSON sizes, but we compress better and don't have costly field names to decode. We also always know the size of everything ahead of our reads, so conversion is greatly simplified.

## Why not Protocol Buffers:

* Protoc - really you don't need much more than that
* Proto2/Proto3 both have features I want and don't want, not a combination of both
	* Proto2 - detect nil vs zero values, but I get pointers everywhere
	* Proto3 - have zero values only except for Message types
* Allocations - proto has to allocate more often, which eventually bogs down GC languages

You might say, why not use `buf.build` tooling for your protoc problems.  I think `buf` is great, but using it in any substantial way requires me to pay. I don't want to figure out how to get my company to pay for their service (it is really easier to roll my own than to navigate what I'm sure would be a bunch of nonsense).  You can hack on `buf cli` tools to let you get around the pay model, but I feel *icky* about that. 

If you need rock solid support for every language and based on an IDL that has been around for years + can figure out how to get your company to pay for it, `buf` is the way to go. 

SERIOUSLY, if you need support of every language and need some rock solid support, go check out `buf.build`. Combine their tools with their `connect.build` and you've got yourself some good stuff there. I'm some guy who spent several years in his free time coming up with the format and the tools. I don't get paid for this and while I will work on bugs, it isn't my job.

Finally, I wanted zero allocations for reads and proto isn't going to provide me that (though Google's internal versions are much better in this regards). Proto is just built on too much legacy stuff and is kind of a mess between proto versions 1 - 3. Now they have releases with various dates, which to me is even worse.

## Why not Capt'n Proto:

Look, that guy is a genius. But the format is hard to understand (I don't know how many times I read his format document before I "got" it).
The RPC mechanism is nuts.  Only he has been able to write an implementation and just for C++ that supports I think more than level 2 support.
Every other language that does simply is loading the C++. And the Go maintainer has left the building the last time I checked.

I don't want to write every implementation of this, so the format has to be easy to understand (at least for people who know how to write binary encoding formats) and the RPC system has to be fairly easy to implement and work with tooling.

## Why not Flat Buffers:

If you make a Message 3 levels deep with some lists, you will figure it out.  Its just hard to use. Also
the Go version doesn't do bounds checks and the code panics on any error. And the performance wasn't really there when I tested it for various use cases I had. However, I had some questions for the author at some point and he was nice/very responsive. 

## Why not Thrift:

I just didn't like the syntax and how it layed out.  I've also never met anyone going to Thrift or pushing Thrift. Nowadays if I mention Thrift, people ask "what is that".

That doesn't say much for it.
