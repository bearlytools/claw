package claw

/*
type Decoder struct {
	mapping Mapping
	mapSize int
}

func NewDecoder(m Mapping) (Decoder, error) {
	return &Decoder{mapping: m}, nil
}

func (d *Decoder) Decode(ctx context.Context, r io.Reader) (Marks, error) {
	m := Marks{
		mapping: d.mapping,
		fields: make(map[uint16][]byte, d.mapSize)
	}

	b64 := pool.get64()

	rd := bufio.NewReader(r)
	_, err := rd.Reader(b64)
	if err != nil {
		return Marks{}, fmt.Errorf("message header missing")
	}

	for {
		_, err := rd.Reader(b64)
		if err != nil && err == io.EOF {
			return m, nil
		}

		field := enc.Uint16(b[0:2])
		desc, ok := d.mapping[field]
		if !ok {
			skip.......
		}
	}
}
*/
