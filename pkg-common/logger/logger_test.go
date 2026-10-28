package logger

import (
	"bytes"
	"strings"
	"testing"
)

func TestNew_StampsAppAndEnv(t *testing.T) {
	var buf bytes.Buffer
	New(Options{App: "svc", Environment: "test", JSON: true, Out: &buf}).Info("hello")

	out := buf.String()
	for _, want := range []string{`"app":"svc"`, `"env":"test"`, `"msg":"hello"`} {
		if !strings.Contains(out, want) {
			t.Errorf("log line %q is missing %s", out, want)
		}
	}
}
