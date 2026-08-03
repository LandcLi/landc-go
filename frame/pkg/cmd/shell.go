package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/LandcLi/landc-go/frame/pkg/trace"
)

// ShellRun 执行 shell 命令，支持链路追踪上下文传递
func ShellRun(ctx context.Context, command string) error {
	return ShellRunWithEnv(ctx, command, nil)
}

// ShellRunWithEnv 执行 shell 命令，支持链路追踪上下文传递和环境变量
func ShellRunWithEnv(ctx context.Context, command string, env []string) error {
	// 创建命令
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	// 设置标准输入输出
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 设置环境变量
	cmd.Env = append(os.Environ(), env...)

	// 传递链路追踪上下文
	traceID := trace.TraceID(ctx)
	spanID := trace.SpanID(ctx)
	parentSpanID := trace.ParentSpanID(ctx)

	if traceID != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TRACE_ID=%s", traceID))
	}
	if spanID != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("SPAN_ID=%s", spanID))
	}
	if parentSpanID != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PARENT_SPAN_ID=%s", parentSpanID))
	}

	// 执行命令
	return cmd.Run()
}

// ShellRunWithOutput 执行 shell 命令并返回输出，支持链路追踪上下文传递
func ShellRunWithOutput(ctx context.Context, command string) (string, error) {
	return ShellRunWithOutputAndEnv(ctx, command, nil)
}

// ShellRunWithOutputAndEnv 执行 shell 命令并返回输出，支持链路追踪上下文传递和环境变量
func ShellRunWithOutputAndEnv(ctx context.Context, command string, env []string) (string, error) {
	// 创建命令
	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	// 设置环境变量
	cmd.Env = append(os.Environ(), env...)

	// 传递链路追踪上下文
	traceID := trace.TraceID(ctx)
	spanID := trace.SpanID(ctx)
	parentSpanID := trace.ParentSpanID(ctx)

	if traceID != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TRACE_ID=%s", traceID))
	}
	if spanID != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("SPAN_ID=%s", spanID))
	}
	if parentSpanID != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PARENT_SPAN_ID=%s", parentSpanID))
	}

	// 执行命令并获取输出
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// RunCommand 执行命令，支持链路追踪上下文传递
func RunCommand(ctx context.Context, name string, args ...string) error {
	return RunCommandWithEnv(ctx, name, args, nil)
}

// RunCommandWithEnv 执行命令，支持链路追踪上下文传递和环境变量
func RunCommandWithEnv(ctx context.Context, name string, args, env []string) error {
	// 创建命令
	cmd := exec.CommandContext(ctx, name, args...)

	// 设置标准输入输出
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 设置环境变量
	cmd.Env = append(os.Environ(), env...)

	// 传递链路追踪上下文
	traceID := trace.TraceID(ctx)
	spanID := trace.SpanID(ctx)
	parentSpanID := trace.ParentSpanID(ctx)

	if traceID != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TRACE_ID=%s", traceID))
	}
	if spanID != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("SPAN_ID=%s", spanID))
	}
	if parentSpanID != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PARENT_SPAN_ID=%s", parentSpanID))
	}

	// 执行命令
	return cmd.Run()
}

// RunCommandWithOutput 执行命令并返回输出，支持链路追踪上下文传递
func RunCommandWithOutput(ctx context.Context, name string, args ...string) (string, error) {
	return RunCommandWithOutputAndEnv(ctx, name, args, nil)
}

// RunCommandWithOutputAndEnv 执行命令并返回输出，支持链路追踪上下文传递和环境变量
func RunCommandWithOutputAndEnv(ctx context.Context, name string, args, env []string) (string, error) {
	// 创建命令
	cmd := exec.CommandContext(ctx, name, args...)

	// 设置环境变量
	cmd.Env = append(os.Environ(), env...)

	// 传递链路追踪上下文
	traceID := trace.TraceID(ctx)
	spanID := trace.SpanID(ctx)
	parentSpanID := trace.ParentSpanID(ctx)

	if traceID != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("TRACE_ID=%s", traceID))
	}
	if spanID != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("SPAN_ID=%s", spanID))
	}
	if parentSpanID != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("PARENT_SPAN_ID=%s", parentSpanID))
	}

	// 执行命令并获取输出
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// InitTraceFromEnv 从环境变量初始化追踪上下文
func InitTraceFromEnv(ctx context.Context) context.Context {
	traceID := os.Getenv("TRACE_ID")
	spanID := os.Getenv("SPAN_ID")
	parentSpanID := os.Getenv("PARENT_SPAN_ID")

	if traceID != "" || spanID != "" || parentSpanID != "" {
		return trace.InitTraceWithParent(ctx, traceID, parentSpanID)
	}

	return ctx
}

// ParseCommand 解析命令字符串为命令名和参数
func ParseCommand(command string) (cmdName string, args []string) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}
