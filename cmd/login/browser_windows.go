//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
)

// openBrowser 用 ShellExecuteW 打开默认浏览器。
// 不经 cmd.exe，URL 里的 & % # 不会被命令行解析，无需转义。
func openBrowser(rawURL string) {
	if os.Getenv(noBrowserEnv) != "" {
		return
	}
	if err := shellExecute(rawURL); err != nil {
		fmt.Fprintf(os.Stderr, "login: 自动打开浏览器失败，请手动复制上面的链接: %v\n", err)
	}
}

func shellExecute(rawURL string) error {
	verb, err := syscall.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := syscall.UTF16PtrFromString(rawURL)
	if err != nil {
		return err
	}
	const swShownormal = 1
	ret, _, _ := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0,
		0,
		swShownormal,
	)
	if ret <= 32 { // ShellExecute 约定：返回值 <= 32 即错误码
		return fmt.Errorf("ShellExecuteW 返回 %d", ret)
	}
	return nil
}
