package config

import (
	"reflect"

	security "github.com/libp2p/go-conn-security"
	crypto "github.com/libp2p/go-libp2p-crypto"
	host "github.com/libp2p/go-libp2p-host"
	pnet "github.com/libp2p/go-libp2p-interface-pnet"
	inet "github.com/libp2p/go-libp2p-net"
	peer "github.com/libp2p/go-libp2p-peer"
	pstore "github.com/libp2p/go-libp2p-peerstore"
	tptu "github.com/libp2p/go-libp2p-transport-upgrader"
	filter "github.com/libp2p/go-maddr-filter"
	mux "github.com/libp2p/go-stream-muxer"
)

var (
	errorType     = reflect.TypeOf((error)(nil))
	upgraderType  = reflect.TypeOf((*tptu.Upgrader)(nil))
	hostType      = reflect.TypeOf((host.Host)(nil))
	networkType   = reflect.TypeOf((inet.Network)(nil))
	muxType       = reflect.TypeOf((mux.Transport)(nil))
	securityType  = reflect.TypeOf((security.Transport)(nil))
	protectorType = reflect.TypeOf((pnet.Protector)(nil))
	filtersType   = reflect.TypeOf((*filter.Filters)(nil))
	peerIDType    = reflect.TypeOf((peer.ID)(""))
	privKeyType   = reflect.TypeOf((crypto.PrivKey)(nil))
	pubKeyType    = reflect.TypeOf((crypto.PubKey)(nil))
	pstoreType    = reflect.TypeOf((pstore.Peerstore)(nil))
)
