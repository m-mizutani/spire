package handler

import (
	"net"
	"strings"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	tld "github.com/jpillora/go-tld"

	"github.com/m-mizutani/gots/set"
	"github.com/m-mizutani/gots/slice"
	"github.com/m-mizutani/spire/pkg/model"
	"github.com/m-mizutani/spire/pkg/service"
	"github.com/m-mizutani/spire/pkg/types"
	"github.com/m-mizutani/spire/pkg/utils"
	"github.com/m-mizutani/ttlcache"
)

type flowHandler struct {
	table   *ttlcache.CacheTable[uint64, *tcpFlow]
	nameMap *service.NameMap
	logCh   chan *model.FlowLog
}

func newFlowHandler(nameMap *service.NameMap, logCh chan *model.FlowLog) *flowHandler {
	hdlr := &flowHandler{
		table:   ttlcache.New[uint64, *tcpFlow](ttlcache.WithExtendByGet()),
		nameMap: nameMap,
		logCh:   logCh,
	}
	hdlr.table.SetHook(func(tf *tcpFlow) uint64 {
		utils.Logger().With("flow", tf.netFlow.String()).Debug("expired")
		if tf.established {
			logCh <- tf.toFlowLog()
		}
		return 0
	})
	return hdlr
}

func (x *flowHandler) Elapse(tick uint64) {
	x.table.Elapse(tick)
}

func (x *flowHandler) ServePacket(ctx *types.Context, pkt gopacket.Packet) {
	net := pkt.NetworkLayer()
	if net == nil {
		return
	}

	tp := pkt.TransportLayer()
	if tp == nil {
		return
	}

	tcpLayer := pkt.Layer(layers.LayerTypeTCP)
	if tcpLayer == nil {
		return
	}
	tcp, ok := tcpLayer.(*layers.TCP)
	if !ok {
		return
	}

	toDomainName := func(fqdn string) string {
		u, err := tld.Parse("https://" + fqdn)
		if err != nil {
			return fqdn
		}
		return strings.Join([]string{u.Domain, u.TLD}, ".")
	}

	flowKey := net.NetworkFlow().FastHash() ^ tp.TransportFlow().FastHash()
	flow := x.table.Get(flowKey)

	if flow == nil {
		srcNames := x.nameMap.LookupNameByAddr(net.NetworkFlow().Src().Raw()).Items()
		dstNames := x.nameMap.LookupNameByAddr(net.NetworkFlow().Dst().Raw()).Items()

		flow = &tcpFlow{
			netFlow: net.NetworkFlow(),
			tpFlow:  tp.TransportFlow(),
			tcp:     tcp,
			state:   tcpInit,
			client: endpoint{
				ip:    net.NetworkFlow().Src().Raw(),
				names: set.New(slice.Map(srcNames, toDomainName)...).Items(),
				port:  int(tcp.SrcPort),
			},
			server: endpoint{
				ip:    net.NetworkFlow().Dst().Raw(),
				names: set.New(slice.Map(dstNames, toDomainName)...).Items(),
				port:  int(tcp.DstPort),
			},
		}

		// first syn packet
		x.table.Set(flowKey, flow, 120)
	}

	sender := flow.getSender(net.NetworkFlow(), tp.TransportFlow())
	sender.dataSize += pkt.Metadata().Length

	if tcp.RST {
		flow.state = tcpClosed
	}

	switch flow.state {
	case tcpInit:
		if tcp.SYN && !tcp.ACK && !tcp.RST && !tcp.FIN {
			flow.state = tcpSynSent
			flow.synAt = pkt.Metadata().Timestamp
		}

	case tcpSynSent:
		if tcp.SYN && tcp.ACK && !tcp.RST && !tcp.FIN {
			flow.state = tcpSynAckSent
			flow.synAckAt = pkt.Metadata().Timestamp
		}

	case tcpSynAckSent:
		if !tcp.SYN && tcp.ACK && !tcp.RST && !tcp.FIN {
			flow.state = tcpEstablished
			flow.established = true
			flow.ackAt = pkt.Metadata().Timestamp
		}

	case tcpEstablished:
		if tcp.FIN {
			if flow.toServer(net.NetworkFlow(), tp.TransportFlow()) {
				utils.Logger().With("flow", flow.netFlow.String()).Debug("close client")
				flow.client.closed = true
			} else {
				utils.Logger().With("flow", flow.netFlow.String()).Debug("close server")
				flow.server.closed = true
			}
		}
	}

	if flow.client.closed && flow.server.closed {
		flow.state = tcpClosed
	}

	if flow.state == tcpClosed {
		x.table.Delete(flowKey)

		flow.closedAt = pkt.Metadata().Timestamp

		if flow.established {
			x.logCh <- flow.toFlowLog()
		}
	}
}

type tcpFlowState int

const (
	tcpInit tcpFlowState = iota + 1
	tcpSynSent
	tcpSynAckSent
	tcpEstablished
	tcpClosed
)

type endpoint struct {
	ip       net.IP
	names    []string
	port     int
	dataSize int
	closed   bool
}

type tcpFlow struct {
	netFlow gopacket.Flow
	tpFlow  gopacket.Flow
	tcp     *layers.TCP

	client      endpoint
	server      endpoint
	state       tcpFlowState
	established bool

	synAt    time.Time
	synAckAt time.Time
	ackAt    time.Time
	closedAt time.Time
}

func (x *tcpFlow) toServer(netFlow, tpFlow gopacket.Flow) bool {
	srcAddr1, _ := x.netFlow.Endpoints()
	srcAddr2, _ := netFlow.Endpoints()
	srcPort1, _ := x.tpFlow.Endpoints()
	srcPort2, _ := tpFlow.Endpoints()

	return srcAddr1 == srcAddr2 && srcPort1 == srcPort2
}

func (x *tcpFlow) getSender(netFlow, tpFlow gopacket.Flow) *endpoint {
	if x.toServer(netFlow, tpFlow) {
		return &x.client
	} else {
		return &x.server
	}
}

func (x *tcpFlow) toFlowLog() *model.FlowLog {
	return &model.FlowLog{
		Client: model.NewEndpoint(
			x.client.ip,
			x.client.port,
			x.client.names,
			x.client.dataSize,
		),
		Server: model.NewEndpoint(
			x.server.ip,
			x.server.port,
			x.server.names,
			x.server.dataSize,
		),
		Latency:  x.ackAt.Sub(x.synAt).Seconds(),
		Duration: x.closedAt.Sub(x.synAt).Seconds(),
	}
}
