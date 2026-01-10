package output

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
)

// ClipboardStrategy はOSのコマンドを利用してクリップボードへ出力する戦略です。
type ClipboardStrategy struct {
	ctx    context.Context // キャンセル制御用
	buffer *bytes.Buffer
}

// NewClipboardStrategy はContextを受け取るように修正されました。
// これにより、プロセス実行時のキャンセル制御が可能になります。
func NewClipboardStrategy(ctx context.Context) *ClipboardStrategy {
	return &ClipboardStrategy{
		ctx:    ctx,
		buffer: new(bytes.Buffer),
	}
}

// Write はメモリ上のバッファに書き込みます。
func (s *ClipboardStrategy) Write(p []byte) (n int, err error) {
	return s.buffer.Write(p)
}

// Close はバッファされた内容をOSのクリップボードコマンドへ流し込みます。
// CommandContextを使用し、Contextキャンセル時は即座に停止します。
func (s *ClipboardStrategy) Close() error {
	// コンテキストが既にキャンセルされている場合は実行しない
	if err := s.ctx.Err(); err != nil {
		return err
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(s.ctx, "pbcopy")
	case "linux":
		// xclipを優先使用
		cmd = exec.CommandContext(s.ctx, "xclip", "-selection", "clipboard")
	case "windows":
		cmd = exec.CommandContext(s.ctx, "clip")
	default:
		return fmt.Errorf("unsupported platform for clipboard: %s", runtime.GOOS)
	}

	cmd.Stdin = s.buffer
	return cmd.Run()
}
