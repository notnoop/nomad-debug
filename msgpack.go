package main

import (
	"bytes"
	"fmt"
	"reflect"
	"time"

	"github.com/hashicorp/go-msgpack/codec"
)

var MsgpackHandle = func() *codec.MsgpackHandle {
	h := &codec.MsgpackHandle{}
	h.RawToString = true
	h.MapType = reflect.TypeOf(map[string]interface{}(nil))
	h.SliceType = reflect.TypeOf([]interface{}(nil))
	return h
}()

func computeTimeSize() int {
	var nb bytes.Buffer
	err := codec.NewEncoder(&nb, MsgpackHandle).Encode(time.Time{})
	if err != nil {
		panic(err)
	}

	// remove one byte container len
	return nb.Len() - 1
}

var timeMsgPackSize = computeTimeSize()

func maybeDecodeTime(v string) (*time.Time, error) {
	if len(v) != timeMsgPackSize {
		return nil, fmt.Errorf("bad length: %d", len(v))
	}

	var nb bytes.Buffer
	err := codec.NewEncoder(&nb, MsgpackHandle).Encode(v)
	if err != nil {
		return nil, err
	}

	var tt time.Time
	err = codec.NewDecoder(&nb, MsgpackHandle).Decode(&tt)
	if err != nil {
		return nil, err
	}

	if tt.IsZero() {
		return nil, nil
	}

	return &tt, nil
}

func fixTime(v interface{}) {
	switch v2 := v.(type) {
	case map[string]interface{}:
		for ek, ev := range v2 {
			if s, ok := ev.(string); ok {
				if t, err := maybeDecodeTime(s); err == nil {
					v2[ek] = t
				}
			} else {
				fixTime(ev)
			}
		}
	case []interface{}:
		for _, e := range v2 {
			fixTime(e)
		}
	default:
		return
	}
}
