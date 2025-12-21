package structwriter

import (
	"bytes"
	"fmt"
	"go/format"
	"go/types"
)

type Struct struct {
	Name     string
	Named    []Field
	Embedded []Field
}

func (s Struct) String() string {
	vars := make([]*types.Var, 0, len(s.Named)+len(s.Embedded))
	for _, emb := range s.Embedded {
		vars = append(vars, types.NewField(0, nil, "", emb.Type, true))
	}
	for _, named := range s.Named {
		vars = append(vars, types.NewField(0, nil, named.Name, named.Type, false))
	}
	st := types.NewStruct(vars, nil)
	b := bytes.Buffer{}
	b.WriteString(fmt.Sprintf("type %s %s", s.Name, st))
	out, err := format.Source(b.Bytes())
	if err != nil {
		panic(err)
	}
	return string(out)
}

type Field struct {
	Name string
	Type types.Type
}
