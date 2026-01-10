package ui

import (
	"errors"
	"io"
	"os"
)

// InputPort は外部からの入力ソース（対話モードの標準入力など）を抽象化します。
type InputPort interface {
	io.Reader

	// Close は入力待ち（ブロッキング）を外部から強制的に解除するために使用します。
	Close() error
}

// ErrInputClosed は入力ポートが閉じられたことを示すエラーです。
var ErrInputClosed = errors.New("input closed")

// StandardInput は os.Stdin をラップし、InputPort インターフェースを実装します。
// io.Pipe を使用することで、os.Stdin 自体を閉じることなく Read を中断可能にします。
type StandardInput struct {
	reader *io.PipeReader
	writer *io.PipeWriter
}

// NewStandardInput は標準入力を使用する StandardInput を作成します。
func NewStandardInput() *StandardInput {
	pr, pw := io.Pipe()
	s := &StandardInput{
		reader: pr,
		writer: pw,
	}

	// ゴルーチンで標準入力を監視し、パイプに流し込みます。
	// アプリケーション終了時（os.Exit）までこのゴルーチンは待機し続ける可能性がありますが、
	// CLIツールの特性上、リソースリークとしては許容範囲内と判断します。
	go s.readStdin()

	return s
}

// readStdin は標準入力からデータを読み込み、パイプに書き込みます。
func (s *StandardInput) readStdin() {
	buf := make([]byte, 1024)
	for {
		n, err := os.Stdin.Read(buf)
		if n > 0 {
			// パイプに書き込む。Reader側が閉じられている場合はエラーになるため終了する。
			if _, wErr := s.writer.Write(buf[:n]); wErr != nil {
				return
			}
		}
		if err != nil {
			s.writer.CloseWithError(err)
			return
		}
	}
}

// Read はパイプからデータを読み込みます。
func (s *StandardInput) Read(p []byte) (n int, err error) {
	return s.reader.Read(p)
}

// Close はパイプの Writer 側をエラー付きで閉じます。
// これにより、Read でブロックしているゴルーチンは即座にエラーを受け取って解除されます。
// os.Stdin 自体は閉じません。
func (s *StandardInput) Close() error {
	return s.writer.CloseWithError(ErrInputClosed)
}
