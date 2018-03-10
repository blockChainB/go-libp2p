package config

import (
	"fmt"
	"reflect"

	host "github.com/libp2p/go-libp2p-host"
	transport "github.com/libp2p/go-libp2p-transport"
	tptu "github.com/libp2p/go-libp2p-transport-upgrader"
	tcp "github.com/libp2p/go-tcp-transport"
	ws "github.com/libp2p/go-ws-transport"
)

// TptC is the type for libp2p transport constructors. You probably won't ever
// implement this function interface directly. Instead, pass your transport
// constructor to TransportConstructor.
type TptC func(h host.Host, u *tptu.Upgrader) (transport.Transport, error)

var transportType = reflect.TypeOf((transport.Transport)(nil))

var transportArgTypes = []struct {
	Type reflect.Type
	New  func(h host.Host, u *tptu.Upgrader) interface{}
}{
	{
		Type: upgraderType,
		New:  func(h host.Host, u *tptu.Upgrader) interface{} { return u },
	},
	{
		Type: hostType,
		New:  func(h host.Host, u *tptu.Upgrader) interface{} { return h },
	},
	{
		Type: networkType,
		New:  func(h host.Host, u *tptu.Upgrader) interface{} { return h.Network() },
	},
	{
		Type: muxType,
		New:  func(h host.Host, u *tptu.Upgrader) interface{} { return u.Muxer },
	},
	{
		Type: securityType,
		New:  func(h host.Host, u *tptu.Upgrader) interface{} { return u.Secure },
	},
	{
		Type: protectorType,
		New:  func(h host.Host, u *tptu.Upgrader) interface{} { return u.Protector },
	},
	{
		Type: filtersType,
		New:  func(h host.Host, u *tptu.Upgrader) interface{} { return u.Filters },
	},
	{
		Type: peerIDType,
		New:  func(h host.Host, u *tptu.Upgrader) interface{} { return h.ID() },
	},
	{
		Type: privKeyType,
		New:  func(h host.Host, u *tptu.Upgrader) interface{} { return h.Peerstore().PrivKey(h.ID()) },
	},
	{
		Type: pubKeyType,
		New:  func(h host.Host, u *tptu.Upgrader) interface{} { return h.Peerstore().PubKey(h.ID()) },
	},
	{
		Type: pstoreType,
		New:  func(h host.Host, u *tptu.Upgrader) interface{} { return h.Peerstore() },
	},
}

// TransportConstructor uses reflection to turn a function that constructs a
// transport into a TptC.
//
// You can pass either a constructed transport (something that implements
// `transport.Transport`) or a function that takes any of:
//
// * The local peer ID.
// * A transport connection upgrader.
// * A private key.
// * A public key.
// * A Host.
// * A Network.
// * A Peerstore.
// * An address filter.
// * A security transport.
// * A stream multiplexer transport.
// * A private network protector.
//
// And returns a type implementing transport.Transport and, optionally, an error
// (as the second argument).
func TransportConstructor(tpt interface{}) (TptC, error) {
	// Already constructed?
	if t, ok := tpt.(transport.Transport); ok {
		return func(_ host.Host, _ *tptu.Upgrader) (transport.Transport, error) {
			return t, nil
		}, nil
	}

	v := reflect.ValueOf(tpt)
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

	argConstructors := make([]func(h host.Host, u *tptu.Upgrader) interface{}, t.NumIn())
outer:
	for i := 0; i < t.NumIn(); i++ {
		argType := t.In(i)
		for _, at := range transportArgTypes {
			if at.Type.Implements(argType) {
				argConstructors[i] = at.New
				break outer
			}
		}
		return nil, fmt.Errorf("argument %d of transport constructor has an unexpected type %s", i, argType.Name())
	}
	return func(h host.Host, u *tptu.Upgrader) (transport.Transport, error) {
		arguments := make([]reflect.Value, len(argConstructors))
		for i, makeArg := range argConstructors {
			arguments[i] = reflect.ValueOf(makeArg(h, u))
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
			return tpt.Interface().(transport.Transport), nil
		default:
			panic("expected 1 or 2 return values from transport constructor")
		}
	}, nil
}

func makeTransports(h host.Host, u *tptu.Upgrader, tpts []TptC) ([]transport.Transport, error) {
	if len(tpts) > 0 {
		transports := make([]transport.Transport, len(tpts))
		for i, tC := range tpts {
			t, err := tC(h, u)
			if err != nil {
				return nil, err
			}
			transports[i] = t
		}
		return transports, nil
	}
	return []transport.Transport{tcp.NewTCPTransport(u), ws.New(u)}, nil
}
