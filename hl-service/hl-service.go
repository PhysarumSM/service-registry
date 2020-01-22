package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-discovery"

	"github.com/Multi-Tier-Cloud/hash-lookup/hl-common"
)

type StringMap struct {
	sync.RWMutex
	data map[string]string
}

func NewStringMap() *StringMap {
	return &StringMap{
		data: make(map[string]string),
	}
}

func (sm *StringMap) Delete(key string) {
	sm.Lock()
	delete(sm.data, key)
	sm.Unlock()
}

func (sm *StringMap) Load(key string) (value string, ok bool) {
	sm.RLock()
	value, ok = sm.data[key]
	sm.RUnlock()
	return value, ok
}

func (sm *StringMap) Store(key, value string) {
	sm.Lock()
	sm.data[key] = value
	sm.Unlock()
}

var nameToHash = NewStringMap()

func handleLookup(stream network.Stream) {
	data, err := common.ReadSingleMessage(stream)
	if err != nil {
		fmt.Println(err)
		return
	}
	
	reqStr := strings.TrimSpace(string(data))
	fmt.Println("Lookup request:", reqStr)
	
	contentHash, ok := nameToHash.Load(reqStr)

	respInfo := common.LookupResponse{contentHash, "", true}
	if !ok {
		respInfo.LookupOk = false
	}

	respBytes, err := json.Marshal(respInfo)
	if err != nil {
		fmt.Println(err)
		stream.Reset()
		return
	}

	err = common.WriteSingleMessage(stream, respBytes)
	if err != nil {
		fmt.Println(err)
		return
	}
}

func handleAdd(stream network.Stream) {
	data, err := common.ReadSingleMessage(stream)
	if err != nil {
		fmt.Println(err)
		return
	}
	
	reqStr := strings.TrimSpace(string(data))
	fmt.Println("Add request:", reqStr)

	var reqInfo common.AddRequest
	err = json.Unmarshal([]byte(reqStr), &reqInfo)
	if err != nil {
		fmt.Println(err)
		stream.Reset()
		return
	}

	nameToHash.Store(reqInfo.ServiceName, reqInfo.ContentHash)
	
	respStr := fmt.Sprintf("Added { %s : %s }",
		reqInfo.ServiceName, reqInfo.ContentHash)
	
	err = common.WriteSingleMessage(stream, []byte(respStr))
	if err != nil {
		fmt.Println(err)
		return
	}
}

func main() {
	nameToHash.Store("hello-there", "test-test-test")

	ctx, host, kademliaDHT, routingDiscovery, err := common.Libp2pSetup(true)
	if err != nil {
		panic(err)
	}
	defer host.Close()
	defer kademliaDHT.Close()

	fmt.Println("Host ID:", host.ID())
	fmt.Println("Listening on:", host.Addrs())

	host.SetStreamHandler(protocol.ID(common.LookupProtocolID), handleLookup)
	host.SetStreamHandler(protocol.ID(common.AddProtocolID), handleAdd)

	discovery.Advertise(ctx, routingDiscovery,
		common.HashLookupRendezvousString)
	fmt.Println("Waiting to serve connections...")

	select {}
}