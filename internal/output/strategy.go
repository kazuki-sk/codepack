package output

import "io"

// Strategy は出力先の抽象化を提供します。
// 仕様書 v1.2.0 準拠:
// ストリーミング処理を前提とし、ドメインモデルやコンテキストには依存せず、
// 標準の io.WriteCloser インターフェースのみを満たします。
type Strategy interface {
	io.WriteCloser
}
