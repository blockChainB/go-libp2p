package config

import (
	"fmt"
	"reflect"

	security "github.com/libp2p/go-conn-security"
	csms "github.com/libp2p/go-conn-security-multistream"
	insecure "github.com/libp2p/go-conn-security/insecure"
	host "github.com/libp2p/go-libp2p-host"
	peer "github.com/libp2p/go-libp2p-peer"
	secio "github.com/libp2p/go-libp2p-secio"
)

// SecC is a security transport constructor
type SecC func(h host.Host) (security.Transport, error)

// MsSecC is a tuple containing a security transport constructor and a protocol
// ID.
type MsSecC struct {
	SecC
	ID string
}

var securityArgTypes = []struct {
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
		Type: privKeyType,
		New:  func(h host.Host) interface{} { return h.Peerstore().PrivKey(h.ID()) },
	},
	{
		Type: pubKeyType,
		New:  func(h host.Host) interface{} { return h.Peerstore().PubKey(h.ID()) },
	},
	{
		Type: pstoreType,
		New:  func(h host.Host) interface{} { return h.Peerstore() },
	},
}

// SecurityConstructor creates a security constructor from the passed parameter
// using reflection.
func SecurityConstructor(sec interface{}) (SecC, error) {
	// Already constructed?
	if t, ok := sec.(security.Transport); ok {
		return func(_ host.Host) (security.Transport, error) {
			return t, nil
		}, nil
	}

	v := reflect.ValueOf(sec)
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
		for _, at := range securityArgTypes {
			if at.Type.Implements(argType) {
				argConstructors[i] = at.New
				break outer
			}
		}
		return nil, fmt.Errorf("argument %d of transport constructor has an unexpected type %s", i, argType.Name())
	}
	return func(h host.Host) (security.Transport, error) {
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
			return tpt.Interface().(security.Transport), nil
		default:
			panic("expected 1 or 2 return values from transport constructor")
		}
	}, nil
}

func makeInsecureTransport(id peer.ID) security.Transport {
	secMuxer := new(csms.SSMuxer)
	secMuxer.AddTransport(insecure.ID, insecure.New(id))
	return secMuxer
}

func makeSecurityTransport(h host.Host, tpts []MsSecC) (security.Transport, error) {
	secMuxer := new(csms.SSMuxer)
	if len(tpts) > 0 {
		transportSet := make(map[string]struct{}, len(tpts))
		for _, tptC := range tpts {
			if _, ok := transportSet[tptC.ID]; ok {
				return nil, fmt.Errorf("duplicate security transport: %s", tptC.ID)
			}
		}
		for _, tptC := range tpts {
			tpt, err := tptC.SecC(h)
			if err != nil {
				return nil, err
			}
			if _, ok := tpt.(*insecure.Transport); ok {
				return nil, fmt.Errorf("cannot construct libp2p with an insecure transport, set the Insecure config option instead")
			}
			secMuxer.AddTransport(tptC.ID, tpt)
		}
	} else {
		id := h.ID()
		sk := h.Peerstore().PrivKey(id)
		secMuxer.AddTransport(secio.ID, &secio.Transport{
			LocalID:    id,
			PrivateKey: sk,
		})
	}
	return secMuxer, nil
}
