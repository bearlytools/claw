I'd like to make a few acknowlegements here. This was built on the shoulders of another who proceeded me and I certainly didn't come up with every idea here.

While the format itself is my own creation, that is only one part of this system.

`Kenton Varda`, the author of Protocol Buffers 2 and CapnProto has certainly influenced this work. My time at Google was full of using proto2/3 for all my services. The author's of the Go version of CapnProto had some great memory management idea, which made influences on how I have handled memory. These ideas attached to the way this format works is responsible for all the speed. 

I also borrowed Packed encoding for use in the RPC system. Because I can simply stream that before going into compression or the wire, it offers a low cost way of shrinking my messages with almost no cost. Our headers can contain lots of zeros, so this worked out for Claw format as well. `Good artists copy, Great artists steal`.

`Go authors`.  The initial versions and later public versions of protos taught me about things I want to do and things I do not want to do. I remember a paper from one of them describing how public accessible fields in protos (vs Getter/Setters) made it near impossible to make enhancements for speed as it would always break some subset of projects. This lead to my use of Getters and Setters. But I also realized that it is not nearly as easly to read doing just Getters/Setters, which is why we have `[Type]Raw` to allow nice methods of creating Claw types.

The Go tooling had a huge impact on what I'm doing here. I want Claw to be as easy to use as the Go tooling and have adopted similar methods to achieve this. No point in re-inventing the wheel.

`VTProto authors`. This extension showed how bad the encoding/decoding of protos is when using reflection and other standard methods. This isn't a comment on the Go proto authors, they had a ton of constraints in the public versions. While I like the extensions, I do not like using things outside the main project. While not as much of a problem, gogo proto was crippled by the second version of Go's proto packages which ended in it being abandoned. That left Kubernetes running an abandoned version of protocol buffers that can't easily be removed.

`Stubby/gRPC teams/Buff Connect`. I borrowed a bunch of RPC syntax and features from what gRPC supports (which is based on Google's Stubby RPC). Like Buff Connect, I also wanted to not be tied to HTTP (TCP is faster in data center fabrics for without the overhead).

Stubby is amazing and has informed everything that came after it. gRPC brought it to the masses. But both tend to use weird parts of HTTP and is generally tied to HTTP2.  Hence I wanted something like gRPC, but with some of the transport choices removed.
