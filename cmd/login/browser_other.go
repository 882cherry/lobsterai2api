//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// openBrowser 调系统命令打开默认浏览器（macOS: open，其他: xdg-open）。
// 失败只提示，不影响登录流程（URL 已打印在 stdout）。
func openBrowser(rawURL string) {
	if os.Getenv(noBrowserEnv) != "" {
		return
	}
	bin := "xdg-open"
	if runtime.GOOS == "darwin" {
		bin = "open"
	}
	cmd := exec.Command(bin, rawURL)
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "login: 自动打开浏览器失败，请手动复制上面的链接: %v\n", err)
		return
	}
	go func() { _ = cmd.Wait() }() // 回收子进程，避免僵死
}
