package pinger

import (
	"fmt"

	"github.com/pomaretta/mcpingserver"
)

type Pinger struct {
	Port int
	S    *mcpingserver.PingServer
}

func New(port int, hook mcpingserver.Responder) *Pinger {
	return &Pinger{
		Port: port,
		S: mcpingserver.CreatePingServer(
			fmt.Sprintf(":%d", port),
			hook,
		),
	}
}

func (p *Pinger) Bind() error {
	return p.S.Bind()
}
