package config

import (
	"fmt"
	"reflect"

	host "github.com/libp2p/go-libp2p-host"
	mux "github.com/libp2p/go-stream-muxer"
	mplex "github.com/whyrusleeping/go-smux-multiplex"
	msmux "github.com/whyrusleeping/go-smux-multistream"
	yamux "github.com/whyrusleeping/go-smux-yamux"
)

// MuxC is a stream multiplex transport constructor
type MuxC func(h host.Host) (mux.Transport, error)

// MsMuxC is a tuple containing a multiplex transport constructor and a protocol
// ID.
type MsMuxC struct {
	MuxC
	ID string
}

var muxArgTypes = []struct {
	Type reflect.Type
	New  func(h host.Host) interface{}
}{
	{
		Type: hostType,
		New:  func(h host.Host) interface{} { return h },
	},
	{
		Type: networkType,
		New:  func(h host.Host) interface{} { return h.Network() },
	},
	{
		Type: peerIDType,
		New:  func(h host.Host) interface{} { return h.ID() },
	},
	{
		Type: pstoreType,
		New:  func(h host.Host) interface{} { return h.Peerstore() },
	},
}

// MuxerConstructor creates a multiplex constructor from the passed parameter
// using reflection.
func MuxerConstructor(m interface{}) (MuxC, error) {
	// Already constructed?
	if t, ok := m.(mux.Transport); ok {
		return func(_ host.Host) (mux.Transport, error) {
			return t, nil
		}, nil
	}

	v := reflect.ValueOf(m)
	t := v.Type()
	if t.Kind() != reflect.Func {
		return nil, fmt.Errorf("expected a transport constructor (function) or a transport")
	}

	switch t.NumOut() {
	case 2:
		if t.Out(1) != errorType {
			return nil, fmt.Errorf("expected (optional) second return value from transport constructor to be an error")
		}

		fallthrough
	case 1:
		if !t.Out(0).Implements(transportType) {
			return nil, fmt.Errorf("expected first return value from transport constructor to be a transport")
		}
	default:
		return nil, fmt.Errorf("expected transport constructor to return a transport and, optionally, an error")
	}

	argConstructors := make([]func(h host.Host) interface{}, t.NumIn())
outer:
	for i := 0; i < t.NumIn(); i++ {
		argType := t.In(i)
		for _, at := range muxArgTypes {
			if at.Type.Implements(argType) {
				argConstructors[i] = at.New
				break outer
			}
		}
		return nil, fmt.Errorf("argument %d of transport constructor has an unexpected type %s", i, argType.Name())
	}
	return func(h host.Host) (mux.Transport, error) {
		arguments := make([]reflect.Value, len(argConstructors))
		for i, makeArg := range argConstructors {
			arguments[i] = reflect.ValueOf(makeArg(h))
		}
		out := v.Call(arguments)
		// Only panics if our reflection logic is bad. The
		// return types have already been checked.
		switch len(out) {
		case 2:
			err := out[1]
			if !err.IsNil() {
				return nil, err.Interface().(error)
			}
			fallthrough
		case 1:
			tpt := out[0]
			return tpt.Interface().(mux.Transport), nil
		default:
			panic("expected 1 or 2 return values from transport constructor")
		}
	}, nil
}

func makeMuxer(h host.Host, tpts []MsMuxC) (mux.Transport, error) {
	muxMuxer := msmux.NewBlankTransport()
	if len(tpts) == 0 {
		transportSet := make(map[string]struct{}, len(tpts))
		for _, tptC := range tpts {
			if _, ok := transportSet[tptC.ID]; ok {
				return nil, fmt.Errorf("duplicate muxer transport: %s", tptC.ID)
			}
		}
		for _, tptC := range tpts {
			tpt, err := tptC.MuxC(h)
			if err != nil {
				return nil, err
			}
			muxMuxer.AddTransport(tptC.ID, tpt)
		}
	} else {
		muxMuxer.AddTransport("/yamux/1.0.0", yamux.DefaultTransport)
		muxMuxer.AddTransport("/mplex/6.3.0", mplex.DefaultTransport)
	}
	return muxMuxer, nil
}
