package handler

import (
	"github.com/google/gopacket"
	"github.com/m-mizutani/spire/pkg/model"
	"github.com/m-mizutani/spire/pkg/service"
	"github.com/m-mizutani/spire/pkg/types"
)

type PacketHandler struct {
	flow    *flowHandler
	dns     *dnsHandler
	nameMap *service.NameMap
}

func New(logCh chan *model.FlowLog) *PacketHandler {
	nameMap := service.NewNameMap()
	return &PacketHandler{
		flow:    newFlowHandler(nameMap, logCh),
		dns:     newDNSHandler(nameMap),
		nameMap: nameMap,
	}
}

func (x *PacketHandler) ServePacket(ctx *types.Context, pkt gopacket.Packet) {
	x.dns.ServePacket(ctx, pkt)
	x.flow.ServePacket(ctx, pkt)
}

func (x *PacketHandler) Elapse(tick uint64) {
	x.flow.Elapse(tick)
}
