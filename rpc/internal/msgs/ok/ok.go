package main

import (
	"context"

	"github.com/bearlytools/claw/rpc/internal/msgs"
)

func main() {
	ctx := context.Background()
	msg := msgs.NewMsg(ctx).SetType(
		msgs.TOpen,
	).SetPayload(
		msgs.NewPayload(ctx).SetReqID(
			32,
		).SetSessionID(8000).SetPayload(
			[]byte("I love the claw format"),
		),
	)

	b, err := msg.Marshal()
	if err != nil {
		panic(err)
	}
	if len(b)%8 != 0 {
		panic("not aligned")
	}

	var result = msgs.NewMsg(ctx)
	if err := result.Unmarshal(b); err != nil {
		panic(err)
	}

	if result.Type() != msg.Type() {
		panic("type mismatch")
	}
	if result.Payload().ReqID() != msg.Payload().ReqID() {
		panic("reqid mismatch")
	}
	if result.Payload().SessionID() != msg.Payload().SessionID() {
		panic("sessionid mismatch")
	}
	if string(result.Payload().Payload()) != string(msg.Payload().Payload()) {
		panic("payload mismatch")
	}
}
