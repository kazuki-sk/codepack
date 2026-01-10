package ui

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

// LargeFileOptions は大容量ファイル処理に必要な設定オプションを定義します。
type LargeFileOptions struct {
	ForceLarge bool
	SkipLarge  bool
}

// Console はCLIにおけるユーザーとの対話を管理します。
type Console struct {
	in     InputPort
	out    io.Writer
	reader *bufio.Reader // レビュー対応: バッファを保持して再利用する
	opts   LargeFileOptions
}

// NewConsole は新しい Console インスタンスを初期化します。
func NewConsole(in InputPort, out io.Writer, opts LargeFileOptions) *Console {
	return &Console{
		in:     in,
		out:    out,
		reader: bufio.NewReader(in), // 初期化時にReaderを作成
		opts:   opts,
	}
}

// Close はConsoleとしてのクリーンアップを行いますが、
// InputPortのClose責務は所有者(main)にあるため、ここでは何もしません。
func (c *Console) Close() error {
	return nil
}

// ShouldInclude は大容量ファイルを処理対象に含めるかどうかをユーザーまたは設定に基づいて判定します。
func (c *Console) ShouldInclude(ctx context.Context, path string, size int64) (bool, error) {
	// 1. 強制包含フラグの確認
	if c.opts.ForceLarge {
		return true, nil
	}

	// 2. 強制除外フラグの確認
	if c.opts.SkipLarge {
		return false, nil
	}

	// 3. 対話的確認
	if err := ctx.Err(); err != nil {
		return false, err
	}

	c.printPrompt(path, size)

	// 保持しているリーダーを使用
	line, err := c.reader.ReadString('\n')

	if err != nil {
		// コンテキストキャンセルが原因のエラーかどうかを確認
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		// InputPort.Close() による意図的な中断の場合
		// レビュー指摘対応: 具体的なエラー型を隠蔽し、キャンセルとして正規化する
		if errors.Is(err, ErrInputClosed) {
			return false, context.Canceled
		}
		return false, err
	}

	input := strings.TrimSpace(strings.ToLower(line))
	if input == "y" || input == "yes" {
		return true, nil
	}

	// デフォルトは No
	return false, nil
}

// printPrompt はユーザーに確認メッセージを表示します。
func (c *Console) printPrompt(path string, size int64) {
	humanSize := formatSize(size)
	fmt.Fprintf(c.out, "\n[?] Large file detected: %s (%s)\n    Include this file? [y/N]: ", path, humanSize)
}

// formatSize はバイトサイズを人間が読みやすい形式に変換します。
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
