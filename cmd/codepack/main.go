package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/kazuki-sk/codepack/internal/config"
	"github.com/kazuki-sk/codepack/internal/ignorer"
	"github.com/kazuki-sk/codepack/internal/language"
	"github.com/kazuki-sk/codepack/internal/output"
	"github.com/kazuki-sk/codepack/internal/processor"
	"github.com/kazuki-sk/codepack/internal/ui"
)

// ldflags で設定されるバージョン情報
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	// 1. 設定のロード
	cfg, err := config.Load(args, os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		return 1
	}

	if cfg.ShowVersion {
		// バージョン表示時にビルド情報も含めるとデバッグ時に有用です
		fmt.Printf("codepack %s (commit: %s, built at: %s)\n", version, commit, date)
		return 0
	}

	// 2. ルートコンテキストとシグナルハンドリングのセットアップ
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3. UIコンポーネントの初期化
	// InputPortのライフサイクル管理はここ(Composition Root)で行う
	inputPort := ui.NewStandardInput()
	
	// キャンセル監視用ゴルーチン
	go func() {
		<-ctx.Done()
		inputPort.Close()
	}()

	largeFileOpts := ui.LargeFileOptions{
		ForceLarge: cfg.ForceLarge,
		SkipLarge:  cfg.SkipLarge,
	}
	console := ui.NewConsole(inputPort, os.Stderr, largeFileOpts)
	
	// 4. Ignorer (除外ロジック) の構築
	ignr := ignorer.NewIgnorer()
	
	// これを最初に行うことで、後続のCLI設定などが優先（または追加）される順序になります。
	if err := ignr.LoadDefaults(); err != nil {
		// 埋め込みリソースの読み込み失敗は致命的な内部エラー
		fmt.Fprintf(os.Stderr, "Error loading default ignore rules: %v\n", err)
		return 1
	}

	// 4.1 CLI パターン (-p) の追加
	if len(cfg.IgnorePatterns) > 0 {
		patternsText := strings.Join(cfg.IgnorePatterns, "\n")
		ignr.AddMatcher(ignorer.NewGitIgnoreMatcher(strings.NewReader(patternsText)))
	}

	// 4.2 CLI 指定ファイル (-i) の追加
	for _, path := range cfg.IgnoreFiles {
		if err := ignr.LoadIgnoreFile(path); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load ignore file '%s': %v\n", path, err)
		}
	}

	// 4.3 ローカル設定 (.code-packignore)
	ignr.LoadIgnoreFile(".code-packignore")

	// 4.4 標準設定 (.gitignore, .dockerignore)
	ignr.LoadIgnoreFile(".gitignore")
	ignr.LoadIgnoreFile(".dockerignore")

	// 5. Language Mapper の初期化
	mapper, err := language.NewMapper(cfg.LanguageMap)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing language mapper: %v\n", err)
		return 1
	}

	// 6. Output Strategy の構築
	var strategies []output.Strategy

	if cfg.OutputFile != "" {
		fileStrategy, err := output.NewFileStrategy(cfg.OutputFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
			return 1
		}
		strategies = append(strategies, fileStrategy)
	}

	if cfg.CopyToClipboard {
		clipStrategy := output.NewClipboardStrategy(ctx)
		strategies = append(strategies, clipStrategy)
	}

	if len(strategies) == 0 {
		fmt.Fprintln(os.Stderr, "Error: No output strategy selected. Please specify -o or -c.")
		return 1
	}

	outStrategy := output.NewMultiStrategy(strategies...)
	defer func() {
		if err := outStrategy.Close(); err != nil && !errors.Is(err, context.Canceled) {
			fmt.Fprintf(os.Stderr, "Error closing output: %v\n", err)
		}
	}()

	// 7. Processor (Core Logic) の初期化と依存注入
	proc, err := processor.NewProcessor(
		cfg.TargetDir,
		cfg.OutputFile,
		ignr,
		mapper,
		outStrategy,
		console, // LargeFileHandlerとして注入
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing processor: %v\n", err)
		return 1
	}

	// 8. 実行
	fmt.Fprintf(os.Stderr, "Packing code from %s...\n", cfg.TargetDir)
	
	if err := proc.Execute(ctx); err != nil {
		// レビュー指摘対応: UI層の内部エラー(ui.ErrInputClosed)への依存を排除し、
		// context.Canceled の判定に統一。
		if errors.Is(err, context.Canceled) {
			fmt.Fprintln(os.Stderr, "\nOperation canceled.")
			return 130
		}
		
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	fmt.Fprintln(os.Stderr, "Done.")
	return 0
}
