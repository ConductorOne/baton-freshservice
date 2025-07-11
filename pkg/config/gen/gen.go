package main

import (
	cfg "github.com/conductorone/baton-freshservice/pkg/config"
	"github.com/conductorone/baton-sdk/pkg/config"
)

func main() {
	config.Generate("freshservice", cfg.Config)
}
