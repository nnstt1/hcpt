package main

import (
	"embed"
	"os"

	"github.com/nnstt1/hcpt/internal/cmd"
)

//go:embed skills
var skillsFS embed.FS

func main() {
	cmd.SetSkillsFS(skillsFS)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
