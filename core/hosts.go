package core

import (
	"strconv"
	"strings"
)

type Host struct {
	Ip   string
	Port int
}

type Hosts struct {
	arg   string
	hosts []Host
}

func ProduceHosts(arg string) *Hosts {
	h := Hosts{}
	h.init(arg)
	return &h
}

func (h *Hosts) init(arg string) {
	arg = strings.TrimSpace(arg)

	sections := strings.Split(arg, ",")
	for i := 0; i < len(sections); i++ {
		h.hosts = append(h.hosts, h.getHost(sections[i]))
	}

}

func (h *Hosts) getHost(arg string) Host {
	sections := strings.Split(arg, ":")
	if len(sections) == 1 {
		return Host{
			Ip:   arg,
			Port: 80,
		}
	}

	port, err := strconv.Atoi(sections[1])
	if err != nil {
		Fatal("parse port in %v failed: %v", arg, err)
	}

	return Host{
		Ip:   sections[0],
		Port: port,
	}
}

func (h *Hosts) GetHosts() []Host {
	return h.hosts
}
