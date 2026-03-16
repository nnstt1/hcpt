package skills

import (
	"io/fs"
	"strings"

	"github.com/Songmu/skillsmith"
	"github.com/spf13/cobra"

	"github.com/nnstt1/hcpt/internal/version"
)

func NewCmdSkills(skillsFS fs.FS) *cobra.Command {
	cmd := &cobra.Command{
		Use:                "skills",
		Short:              "Manage agent skills for AI coding assistants",
		DisableFlagParsing: true,
		SilenceUsage:       true,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := skillsmith.New("hcpt", sanitizeVersion(version.Version), skillsFS)
			if err != nil {
				return err
			}
			return s.Run(cmd.Context(), args)
		},
	}
	return cmd
}

// sanitizeVersion extracts a clean semver string from the version.
// version.Version may contain extra info like "v0.8.0 (abc1234)" or "dev".
func sanitizeVersion(v string) string {
	// Take only the first space-delimited token: "v0.8.0 (abc1234)" -> "v0.8.0"
	if i := strings.IndexByte(v, ' '); i > 0 {
		v = v[:i]
	}
	// Strip "v" prefix if present, and any "+dirty" suffix
	v = strings.TrimPrefix(v, "v")
	if i := strings.Index(v, "+"); i > 0 {
		v = v[:i]
	}
	// If it's not semver-like (e.g. "dev"), use a fallback
	if !strings.Contains(v, ".") {
		return "0.0.0-dev"
	}
	return v
}
