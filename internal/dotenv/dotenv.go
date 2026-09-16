// Package dotenv 从 .env 文件加载环境变量，省去每次启动前在终端 export。
//
// 只依赖标准库；进程里已存在的环境变量优先，.env 只是"默认值"。
package dotenv

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	// FileEnvVar 指定 .env 路径的环境变量名（显式指定时找不到即报错）。
	FileEnvVar = "LB2A_ENV_FILE"
	// DefaultName 默认文件名。
	DefaultName = ".env"
)

// Load 逐行解析 path 并把键值写入进程环境变量。
//
// 语法：空行与 # 开头的行为注释；允许 `export KEY=VALUE`；值两端成对的引号会被去掉。
// 同一文件内后出现的键覆盖先出现的；进程里已存在的键不覆盖（export 的命令行变量优先）。
// 文件不存在时返回 os.ErrNotExist 包装的错误。
func Load(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fromFile := map[string]bool{}
	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		key, val, ok, err := parseLine(sc.Text())
		if err != nil {
			return fmt.Errorf("%s:%d: %w", path, line, err)
		}
		if !ok {
			continue
		}
		if !fromFile[key] {
			if _, exists := os.LookupEnv(key); exists {
				continue // 真实环境变量优先
			}
		}
		if err := os.Setenv(key, val); err != nil {
			return fmt.Errorf("%s:%d: setenv %s: %w", path, line, key, err)
		}
		fromFile[key] = true
	}
	return sc.Err()
}

// parseLine 解析一行；ok 为 false 表示该行应被忽略。
func parseLine(line string) (key, val string, ok bool, err error) {
	const bom = "\ufeff" // 记事本另存为 UTF-8 时可能带 BOM
	s := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), bom))
	if s == "" || strings.HasPrefix(s, "#") {
		return "", "", false, nil
	}
	s = strings.TrimSpace(strings.TrimPrefix(s, "export "))
	i := strings.IndexByte(s, '=')
	if i < 0 {
		return "", "", false, fmt.Errorf("want KEY=VALUE, got %q", s)
	}
	key = strings.TrimSpace(s[:i])
	val = strings.TrimSpace(s[i+1:])
	if key == "" {
		return "", "", false, fmt.Errorf("empty key in %q", s)
	}
	if n := len(val); n >= 2 {
		if (val[0] == '"' && val[n-1] == '"') || (val[0] == '\'' && val[n-1] == '\'') {
			val = val[1 : n-1]
		}
	}
	return key, val, true, nil
}

// LoadDefault 查找并加载 .env，返回实际加载的路径（没找到返回 ""）。
//
// 查找顺序：
//  1. $LB2A_ENV_FILE（显式指定，文件不存在/格式错误都直接报错）
//  2. 当前工作目录下的 .env
//  3. 可执行文件同目录下的 .env（Windows 双击 exe、注册成服务/计划任务时工作目录不可靠）
func LoadDefault() (string, error) {
	if p := os.Getenv(FileEnvVar); p != "" {
		return p, Load(p)
	}
	candidates := []string{DefaultName}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), DefaultName))
	}
	for _, p := range candidates {
		err := Load(p)
		if err == nil {
			return p, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return p, err // 文件存在但格式有误：报错，别静默忽略
		}
	}
	return "", nil
}
