package wake

import (
	"fmt"
	"log"
	"net"
	"time"

	pinging "github.com/go-ping/ping"
	"github.com/mdlayher/wol"
	"github.com/pomaretta/mcpingserver"
)

type WakeTarget struct {
	Ma net.HardwareAddr
	Ba net.IP
	Ip net.IP
	E  bool
}

type WakeResponder struct {
	r  *mcpingserver.PingResponse
	kr interface{}
	l  *mcpingserver.LegacyPingResponse

	lst    time.Time
	target *WakeTarget
}

func New(r *mcpingserver.PingResponse, kr interface{}, l *mcpingserver.LegacyPingResponse, target *WakeTarget) *WakeResponder {
	return &WakeResponder{
		r:      r,
		kr:     kr,
		l:      l,
		lst:    time.Time{},
		target: target,
	}
}

func (wr *WakeResponder) OnConnect(a net.Addr) error {
	return nil
}

func (wr *WakeResponder) RespondPing(h *mcpingserver.Handshake) (*mcpingserver.PingResponse, error) {
	log.Printf("[PING] Responding ping to %s", h.SourceIP.String())
	return wr.r, nil
}

func (wr *WakeResponder) RespondJoin(h *mcpingserver.Handshake, uid string) (interface{}, error) {
	log.Printf(
		"[JOIN] Attempt from %s, uid=%s, lst=%s",
		h.SourceIP.String(),
		uid,
		wr.lst.String(),
	)
	if time.Since(wr.lst) < time.Minute*5 {
		log.Printf("[JOIN] %s is too soon", h.SourceIP.String())
		return "Hace menos de 5 minutos que se ha mandado el mensaje, espera un poco. - \u00A7b\u00A7l\u00A7nBernardo el Goblin", nil
	}

	if !wr.target.E {
		log.Printf("[JOIN] %s joined with %s address, wol is disabled", uid, h.SourceIP.String())
		return "El encendido automático está deshabilitado, vuelve más tarde - \u00A7b\u00A7l\u00A7nBernardo el Goblin", nil
	}

	ok := wr.isAlive()
	if ok {
		log.Printf("[JOIN] %s is alive, returning kr", wr.target.Ip.String())
		return wr.kr, nil
	}
	c, err := wol.NewClient()
	if err != nil {
		log.Printf("[JOIN] Error creating wol client: %s", err)
		return "Ha ocurrido un problema al crear el cliente de WOL, contacta con \u00A7b\u00A7lpomaretta. - \u00A7b\u00A7l\u00A7nBernardo el Goblin", nil
	}
	log.Printf("[JOIN] Sending WOL to %s using MAC %s", wr.target.Ba.String(), wr.target.Ma.String())
	err = c.Wake(
		fmt.Sprintf("%s:%d", wr.target.Ba.String(), 9),
		wr.target.Ma,
	)
	if err != nil {
		log.Printf("[JOIN] Error sending wol packet: %s", err)
		return "Ha ocurrido un problema al enviar el mensaje de WOL, contacta con \u00A7b\u00A7lpomaretta. - \u00A7b\u00A7l\u00A7nBernardo el Goblin", nil
	}
	wr.lst = time.Now()
	log.Printf("[JOIN] Wake sent to %s at %s", wr.target.Ip.String(), wr.lst.String())
	return fmt.Sprintf(
		"Hola \u00A7d\u00A7l%s\u00A7r, soy \u00A7b\u00A7l\u00A7nBernardo el Goblin\u00A7r, he enviado el mensaje a nuestro compañero \u00A76\u00A7l%s\u00A7r, ¡espera un momento!",
		uid,
		"cucumber.pomaretta.com",
	), nil
}

func (wr *WakeResponder) RespondLegacyPing(h *mcpingserver.Handshake) (*mcpingserver.LegacyPingResponse, error) {
	log.Printf("[LEGACY PING] Responding legacy ping to %s", h.SourceIP.String())
	return wr.l, nil
}

func (wr *WakeResponder) isAlive() bool {
	pinger, err := pinging.NewPinger(wr.target.Ip.String())
	if err != nil {
		return false
	}
	pinger.SetPrivileged(true)
	pinger.Count = 1
	pinger.Interval = time.Second
	pinger.Timeout = time.Second * 5
	pinger.Run()
	return pinger.PacketsRecv != 0
}
