package output

import (
	"bufio"
	"errors"
	"os"
)

// FileStrategy は指定されたパスへファイルを出力する戦略です。
type FileStrategy struct {
	file   *os.File
	writer *bufio.Writer
}

// NewFileStrategy は新しいFileStrategyを初期化します。
func NewFileStrategy(path string) (*FileStrategy, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &FileStrategy{
		file:   f,
		writer: bufio.NewWriter(f),
	}, nil
}

// Write はバッファリングされた書き込みを行います。
func (s *FileStrategy) Write(p []byte) (n int, err error) {
	return s.writer.Write(p)
}

// Close はバッファをフラッシュし、ファイルを閉じます。
// FlushエラーとCloseエラーの両方を捕捉します。
func (s *FileStrategy) Close() error {
	flushErr := s.writer.Flush()
	closeErr := s.file.Close()
	
	// Go 1.20+ errors.Join を使用してエラーを合成
	return errors.Join(flushErr, closeErr)
}
