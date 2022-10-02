package handler

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/m-mizutani/spire/pkg/service"
	"github.com/m-mizutani/spire/pkg/types"
)

type dnsHandler struct {
	nameMap *service.NameMap
}

func newDNSHandler(nameMap *service.NameMap) *dnsHandler {
	return &dnsHandler{
		nameMap: nameMap,
	}
}

func (x *dnsHandler) ServePacket(ctx *types.Context, pkt gopacket.Packet) {
	dnsLayer := pkt.Layer(layers.LayerTypeDNS)
	if dnsLayer == nil {
		return
	}
	dns, ok := dnsLayer.(*layers.DNS)
	if !ok {
		return
	}

	for _, answer := range dns.Answers {
		switch answer.Type {
		case layers.DNSTypeA:
			var key [4]byte
			copy(key[:], answer.Data[0:4])
			x.nameMap.InsertNameWithV4(key, string(answer.Name), answer.TTL)

		case layers.DNSTypeAAAA:
			var key [16]byte
			copy(key[:], answer.Data[:])
			x.nameMap.InsertNameWithV6(key, string(answer.Name), answer.TTL)

		case layers.DNSTypeCNAME:
			x.nameMap.InsertCName(string(answer.Name), string(answer.CNAME), answer.TTL)
		}
	}
}
