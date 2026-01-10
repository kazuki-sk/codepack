package output

import (
	"io"
)

// MultiStrategy は複数のStrategyに対して同時に書き込みを行います。
type MultiStrategy struct {
	strategies []Strategy
}

// NewMultiStrategy は複数の出力先をまとめます。
func NewMultiStrategy(strategies ...Strategy) Strategy {
	return &MultiStrategy{strategies: strategies}
}

// Write は全てのStrategyに書き込みます。
func (m *MultiStrategy) Write(p []byte) (n int, err error) {
	for _, s := range m.strategies {
		n, err = s.Write(p)
		if err != nil {
			return n, err
		}
		if n != len(p) {
			return n, io.ErrShortWrite
		}
	}
	return len(p), nil
}

// Close は全てのStrategyを閉じます。エラーは最初の一つを返します。
func (m *MultiStrategy) Close() error {
	var firstErr error
	for _, s := range m.strategies {
		if err := s.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
