package main

import (
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/BurntSushi/toml"
	"github.com/pomaretta/mcpingserver"
	"github.com/pomaretta/mcwake/pinger"
	"github.com/pomaretta/mcwake/pinger/wake"
)

type Config struct {
	Port int

	Motd          string
	MaxPlayers    int
	OnlinePlayers int
	KickResponse  string

	ServerVersion   string
	ProtocolVersion int

	TargetMac       string
	TargetBroadcast string
	TargetIp        string
	Enabled         bool
}

type PingerWrapper struct {
	Exit   bool
	Active bool
	Pinger *pinger.Pinger
}

func readConfig() (*Config, error) {

	var cp string
	var config Config

	flag.StringVar(&cp, "c", "configuration.conf", "Path to config file")
	flag.Parse()

	if _, err := toml.DecodeFile(cp, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func main() {

	config, err := readConfig()
	if err != nil {
		panic(err)
	}

	// NOTE: Parse MAC address
	mac, err := net.ParseMAC(config.TargetMac)
	if err != nil {
		panic(err)
	}

	// NOTE: Parse IP address
	ip := net.ParseIP(config.TargetIp)
	if ip == nil {
		panic("Invalid IP address")
	}

	// NOTE: Parse broadcast address
	broadcast := net.ParseIP(config.TargetBroadcast)
	if broadcast == nil {
		panic("Invalid broadcast address")
	}

	hook := wake.New(
		&mcpingserver.PingResponse{
			Description: config.Motd,
			Players: mcpingserver.PlayersEntry{
				MaxPlayers:    config.MaxPlayers,
				OnlinePlayers: config.OnlinePlayers,
			},
			Version: mcpingserver.VersionEntry{
				Name:     config.ServerVersion,
				Protocol: uint(config.ProtocolVersion),
			},
		},
		config.KickResponse,
		&mcpingserver.LegacyPingResponse{
			Motd:            config.Motd,
			PlayerCount:     config.OnlinePlayers,
			PlayerMax:       config.MaxPlayers,
			ProtocolVersion: config.ProtocolVersion,
			ServerVersion:   config.ServerVersion,
		},
		&wake.WakeTarget{
			Ma: mac,
			Ba: broadcast,
			Ip: ip,
			E:  config.Enabled,
		},
	)

	log.Printf("[MAIN] Starting server on port %d", config.Port)
	p := pinger.New(
		config.Port,
		hook,
	)
	p.S.SetResponseTimeout(0)

	pw := PingerWrapper{
		Active: false,
		Pinger: p,
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGHUP)
	go func() {
		for {
			sig := <-sigs
			switch sig {
			case syscall.SIGHUP:
				log.Println("SIGHUP received, reloading")
				pw.Pinger.S.Close()
				pw.Active = false
			default:
				log.Println("Signal received", sig)
				pw.Exit = true
				pw.Active = false
			}
		}
	}()

	go func() {
		for !pw.Exit {
			if !pw.Active {
				continue
			}
			p.S.AcceptConnection(func(err error) {
				if err == nil {
					return
				}
				err = p.S.Close()
				if err != nil {
					log.Println("[ERROR] Failed to close server:", err)
				}
				log.Println("[MAIN] Listener closed.")
				log.Println("[MAIN] Recovering server.")
				pw.Active = false
			})
		}
	}()

	for !pw.Exit {
		if pw.Active {
			continue
		}
		err = pw.Pinger.Bind()
		if err != nil {
			log.Printf("[MAIN] Error: %s", err)
			continue
		}
		pw.Active = true
		log.Printf("[MAIN] Listener started.")
	}

}
