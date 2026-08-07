package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/LandcLi/landc-go/frame/pkg/cmd"
)

// NewDoctorCommand 创建 doctor 环境诊断命令。
// 主动检查 landc-go 开发环境，命中常见问题：
//   - go 工具链缺失或版本低于框架要求（frame 需要 Go 1.26+）
//   - GOPROXY 为空/off（依赖拉取必失败）
//   - 依赖模块在 GOPROXY 中不可解析（内网代理未同步 landc-go）
//   - 当前目录非 Go 项目 / 缺少 config.yaml
//
// 可选 --check-network 会发起 go list -m 网络查询（默认离线检查，避免卡顿）。
func NewDoctorCommand() *cmd.Command {
	command := cmd.NewCommand("doctor", "Diagnose environment for landc-go development", func(ctx context.Context, parser *cmd.Parser) error {
		ok := true

		fmt.Printf("=== landc doctor ===\n")

		// 1. landc 自身版本
		fmt.Printf("[landc] %s\n", cliVersion())

		// 2. go 工具链
		goVer, err := exec.Command("go", "version").Output()
		if err != nil {
			fmt.Printf("[FAIL] `go` not found in PATH: %v\n", err)
			fmt.Println("  -> 请先安装 Go 1.26+：https://go.dev/dl/")
			ok = false
		} else {
			ver := strings.TrimSpace(string(goVer))
			fmt.Printf("[go] %s（landc 编译于 %s）\n", ver, runtime.Version())
			if !goVersionOK(ver) {
				fmt.Println("  -> 版本低于 1.26，frame 模块需要 Go 1.26+，请升级工具链")
				ok = false
			} else {
				fmt.Println("  -> 版本满足要求")
			}
		}

		// 3. GOPROXY
		proxy, _ := exec.Command("go", "env", "GOPROXY").Output()
		proxyStr := strings.TrimSpace(string(proxy))
		fmt.Printf("[goproxy] %s\n", proxyStr)
		if proxyStr == "" || proxyStr == "off" {
			fmt.Println("  -> GOPROXY 为空或 off，模块拉取将失败；建议设为 https://proxy.golang.org,direct")
			ok = false
		} else if strings.Contains(proxyStr, "off") {
			fmt.Println("  -> 注意：GOPROXY 含 off，部分路径可能无法拉取")
		}

		// 4. 当前目录是否为 Go 项目
		if data, err := os.ReadFile("go.mod"); err == nil {
			mod := moduleNameFromGoMod(data)
			fmt.Printf("[module] %s\n", mod)
		} else {
			fmt.Println("[module] 当前目录未找到 go.mod（非 Go 项目，或需 cd 到项目目录）")
		}

		// 5. config.yaml
		if _, err := os.Stat("config.yaml"); err == nil {
			fmt.Println("[config] config.yaml 存在")
		} else {
			fmt.Println("[config] 未找到 config.yaml（landc init 生成的项目会有；手动搭建可忽略）")
		}

		// 6. 依赖可解析性（网络检查，默认关闭）
		if parser.HasOpt("check-network") {
			out, err := exec.Command("go", "list", "-m", "-versions", "github.com/LandcLi/landc-go/frame").CombinedOutput()
			if err != nil {
				fmt.Printf("[FAIL] 无法解析 github.com/LandcLi/landc-go/frame：%v\n%s", err, out)
				fmt.Println("  -> 请检查 GOPROXY 是否可访问；内网代理需先同步 landc-go 各模块")
				ok = false
			} else {
				fmt.Printf("[network] 可解析：%s\n", strings.TrimSpace(string(out)))
			}
		}

		fmt.Println()
		if ok {
			fmt.Println("✓ 环境基本健康")
			return nil
		}
		return fmt.Errorf("environment has issues, see output above")
	})
	command.AddOption("check-network", false)
	return command
}

// goVersionOK 解析 `go version go1.26.5 darwin/arm64` 并判断是否 >= 1.26。
func goVersionOK(goVer string) bool {
	fields := strings.Fields(goVer)
	if len(fields) < 3 || !strings.HasPrefix(fields[2], "go") {
		return false
	}
	v := strings.TrimPrefix(fields[2], "go")
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return false
	}
	major, err1 := strconv.Atoi(parts[0])
	minor, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil {
		return false
	}
	if major > 1 {
		return true
	}
	return major == 1 && minor >= 26
}

// moduleNameFromGoMod 提取 go.mod 的 module 声明。
func moduleNameFromGoMod(data []byte) string {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return "unknown"
}
