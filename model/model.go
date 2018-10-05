package model

import (
	"time"
)

// NodeConfig Structure of node config
type NodeConfig struct {
	ID                   string        `json:"id"`
	IP                   string        `json:"ip"`
	Port                 int           `json:"port"`
	DisseminationTimeout time.Duration `json:"dissemination_timeout"`
}
