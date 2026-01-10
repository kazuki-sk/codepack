package processor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kazuki-sk/codepack/internal/ignorer"
	"github.com/kazuki-sk/codepack/internal/language"
	"github.com/kazuki-sk/codepack/internal/output"
)

// DefaultThreshold は大容量ファイルとみなす閾値（500KB）です。
const DefaultThreshold = 500 * 1024

// Processor はファイルシステムの走査とコンテンツ処理を行うコアロジックです。
type Processor struct {
	targetDir        string
	absOutputPath    string // 自己参照防止用の絶対パス
	ignorer          *ignorer.Ignorer
	mapper           *language.Mapper
	output           output.Strategy
	largeFileHandler LargeFileHandler
}

// NewProcessor はProcessorを初期化します。
// outputFile: 出力先のファイルパス（自己参照除外判定に使用）
func NewProcessor(
	targetDir string,
	outputFile string,
	ignr *ignorer.Ignorer,
	mpr *language.Mapper,
	out output.Strategy,
	lfh LargeFileHandler,
) (*Processor, error) {
	// 出力ファイルの絶対パスを解決して保持（存在しなくてもパス比較は可能）
	absOut, err := filepath.Abs(outputFile)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve output file path: %w", err)
	}

	return &Processor{
		targetDir:        targetDir,
		absOutputPath:    absOut,
		ignorer:          ignr,
		mapper:           mpr,
		output:           out,
		largeFileHandler: lfh,
	}, nil
}

// Execute は対象ディレクトリの走査とMarkdown生成を実行します。
//
// Note: 本メソッドは `Output Strategy` への書き込み完了までを責務としますが、
// Outputの `Close` (Flush) 処理は呼び出し元（main）の責務です。
func (p *Processor) Execute(ctx context.Context) error {
	err := filepath.WalkDir(p.targetDir, func(path string, d fs.DirEntry, err error) error {
		// 1. キャンセルチェック: ユーザーの中断シグナルを検知したら即座に終了
		if err := ctx.Err(); err != nil {
			return err
		}

		if err != nil {
			// アクセス権限エラーなどはスキップして続行
			// ログ機構があればここでWarnログを出力
			return nil
		}

		// 2. ディレクトリの処理
		if d.IsDir() {
			if p.ignorer.ShouldIgnore(path, true) {
				return filepath.SkipDir
			}
			return nil
		}

		// 3. シンボリックリンクのスキップ（仕様 3.3）
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		// 4. 自己参照チェック（仕様 3.2/3.3）
		// 出力ファイル自体を読み込まないように除外
		absPath, err := filepath.Abs(path)
		if err == nil && absPath == p.absOutputPath {
			return nil
		}

		// 5. ファイルの除外判定
		if p.ignorer.ShouldIgnore(path, false) {
			return nil
		}

		// 6. ファイル処理の実行
		return p.processFile(ctx, path, info)
	})

	return err
}

// processFile は単一ファイルの読み込み、判定、出力を行います。
func (p *Processor) processFile(ctx context.Context, path string, info fs.FileInfo) error {
	// ファイルオープン
	file, err := os.Open(path)
	if err != nil {
		return nil // 読み込み不可ファイルはスキップ
	}
	defer file.Close()

	// オープン直後にもキャンセルチェック（待機中にキャンセルされた場合など）
	if err := ctx.Err(); err != nil {
		return err
	}

	// A. バイナリ判定（仕様準拠: io.LimitReader使用）
	// 先頭512バイトまでを読み込む。512バイト未満の場合はEOFまでのデータが返る。
	headBuf, err := io.ReadAll(io.LimitReader(file, 512))
	if err != nil {
		return nil
	}

	if isBinary(headBuf) {
		// バイナリの場合はパスのみ記録（プレースホルダー出力）
		return p.writeMarkdownEntry(ctx, path, "", nil, true)
	}

	// B. サイズ制限判定
	if info.Size() > DefaultThreshold {
		include, err := p.largeFileHandler.ShouldInclude(ctx, path, info.Size())
		if err != nil {
			return err
		}
		if !include {
			return nil // ユーザーまたは設定により除外
		}
	}

	// C. コンテンツ出力
	// 読み込んだheadBufと、続きのfileストリームを結合して渡す
	reader := io.MultiReader(bytes.NewReader(headBuf), file)
	lang := p.mapper.GetLanguage(path)

	return p.writeMarkdownEntry(ctx, path, lang, reader, false)
}

// writeMarkdownEntry はMarkdown形式のフォーマットと出力を行います。
// 本来はOutput層の責務ですが、既存インターフェース制約のためここでフォーマットし、
// Output Strategyにはバイトストリームとして書き込みます。
func (p *Processor) writeMarkdownEntry(ctx context.Context, path, lang string, r io.Reader, isBinarySkipped bool) error {
	// パス区切り文字の統一（仕様 3.3）
	relPath, err := filepath.Rel(p.targetDir, path)
	if err != nil {
		relPath = path
	}
	normalizedPath := filepath.ToSlash(relPath)

	var header string
	if isBinarySkipped {
		header = fmt.Sprintf("\n## File: %s\n\n(Binary file skipped)\n", normalizedPath)
	} else {
		header = fmt.Sprintf("\n## File: %s\n\n```%s\n", normalizedPath, lang)
	}

	// ヘッダー書き込み
	if _, err := p.output.Write([]byte(header)); err != nil {
		return err
	}

	// コンテンツ書き込み（バイナリスキップでない場合）
	if !isBinarySkipped && r != nil {
		if err := p.copyCancellable(ctx, p.output, r); err != nil {
			return err
		}
		// フッター書き込み
		if _, err := p.output.Write([]byte("\n```\n")); err != nil {
			return err
		}
	}

	return nil
}

// copyCancellable は io.Copy の代わりに使用し、Contextのキャンセルを検知しながらコピーを行います。
// これにより、大容量ファイル書き込み中の即時中断が可能になります。
func (p *Processor) copyCancellable(ctx context.Context, dst io.Writer, src io.Reader) error {
	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 読み込み
			nr, er := src.Read(buf)
			if nr > 0 {
				// 書き込み
				nw, ew := dst.Write(buf[0:nr])
				if ew != nil {
					return ew
				}
				if nr != nw {
					return io.ErrShortWrite
				}
			}
			if er != nil {
				if er == io.EOF {
					return nil
				}
				return er
			}
		}
	}
}

// isBinary はバイト列からバイナリかどうかを判定します。
func isBinary(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	// NULLバイトが含まれる場合はバイナリとみなす
	if bytes.IndexByte(data, 0) != -1 {
		return true
	}
	// http.DetectContentTypeによる判定
	contentType := http.DetectContentType(data)
	return !strings.HasPrefix(contentType, "text/") &&
		contentType != "application/json" &&
		contentType != "application/xml"
}
