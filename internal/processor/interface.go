package processor

import (
	"context"
)

// LargeFileHandler は大容量ファイルの扱いを決定するポリシーインターフェースです。
// 仕様書 v1.2.0 準拠: Coreは「処理するか否か」のみに関心を持ちます。
type LargeFileHandler interface {
	// ShouldInclude は対象のファイルを処理対象に含めるべきか判定します。
	// ctx: キャンセル伝播用
	// path: ファイルパス
	// size: ファイルサイズ
	// 戻り値: (true=含める/false=除外, エラー)
	ShouldInclude(ctx context.Context, path string, size int64) (bool, error)
}
