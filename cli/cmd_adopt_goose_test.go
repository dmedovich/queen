package cli

import (
	"strings"
	"testing"
)

func TestAdoptGooseRequiresPreviewTokenAndSchemaVerification(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--apply"}, "--plan"},
		{[]string{"--apply", "--plan", "abc"}, "--verified-schema"},
		{[]string{"--verified-schema"}, "valid only with --apply"},
		{[]string{"--plan", "abc"}, "valid only with --apply"},
	} {
		app := &App{config: &Config{}}
		cmd := app.adoptGooseCmd()
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("args %v: error = %v, want %q", tc.args, err, tc.want)
		}
	}
}
