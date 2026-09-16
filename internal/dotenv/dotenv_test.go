package dotenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseLine(t *testing.T) {
	cases := []struct {
		in      string
		key     string
		val     string
		ok      bool
		wantErr bool
	}{
		{in: "", ok: false},
		{in: "   ", ok: false},
		{in: "# comment", ok: false},
		{in: "  # indented comment", ok: false},
		{in: "LB2A_LISTEN=:8367", key: "LB2A_LISTEN", val: ":8367", ok: true},
		{in: "  KEY = spaced value  ", key: "KEY", val: "spaced value", ok: true},
		{in: "export KEY=v", key: "KEY", val: "v", ok: true},
		{in: `KEY="quoted value"`, key: "KEY", val: "quoted value", ok: true},
		{in: "KEY='sq'", key: "KEY", val: "sq", ok: true},
		{in: "URL=https://h/p#frag", key: "URL", val: "https://h/p#frag", ok: true},
		{in: "EMPTY=", key: "EMPTY", val: "", ok: true},
		{in: "\ufeffKEY=bom", key: "KEY", val: "bom", ok: true},
		{in: "KEY=v\r", key: "KEY", val: "v", ok: true}, // CRLF
		{in: "NOEQUALS", wantErr: true},
		{in: "=v", wantErr: true},
	}
	for _, c := range cases {
		key, val, ok, err := parseLine(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseLine(%q): want error, got none", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseLine(%q): %v", c.in, err)
			continue
		}
		if ok != c.ok || key != c.key || val != c.val {
			t.Errorf("parseLine(%q) = (%q, %q, %v), want (%q, %q, %v)",
				c.in, key, val, ok, c.key, c.val, c.ok)
		}
	}
}

func TestLoadPrecedence(t *testing.T) {
	t.Setenv("EXISTING", "from-process")
	// t.Setenv 登记"用例结束后恢复原值"，先设再删，确保本用例内 DUP 是未设置的
	t.Setenv("DUP", "")
	os.Unsetenv("DUP")

	path := filepath.Join(t.TempDir(), ".env")
	content := "# comment\n" +
		"EXISTING=from-file\n" + // 进程里已有 → 保留进程的值
		"DUP=first\n" +
		"DUP=second\n" + // 文件内后写的覆盖先写的
		"NEW=\"quoted\"\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Load(path); err != nil {
		t.Fatal(err)
	}

	if got := os.Getenv("EXISTING"); got != "from-process" {
		t.Errorf("EXISTING = %q, want from-process (真实环境变量优先)", got)
	}
	if got := os.Getenv("DUP"); got != "second" {
		t.Errorf("DUP = %q, want second (同一文件内后写覆盖)", got)
	}
	if got := os.Getenv("NEW"); got != "quoted" {
		t.Errorf("NEW = %q, want quoted", got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	err := Load(filepath.Join(t.TempDir(), "nope.env"))
	if !os.IsNotExist(err) {
		t.Errorf("Load(missing) = %v, want IsNotExist", err)
	}
}

func TestLoadBadLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GOOD=1\nBROKEN\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Load(path); err == nil {
		t.Fatal("Load(bad line): want error, got none")
	}
}
