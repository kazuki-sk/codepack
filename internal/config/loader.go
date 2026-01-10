package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
)

// arrayFlags はフラグで複数回指定可能な文字列スライスを扱います。
type arrayFlags []string

func (i *arrayFlags) String() string {
	return fmt.Sprint(*i)
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// Load はコマンドライン引数を解析します。
// UI層への依存を減らすため、Usageの定義は最小限、または呼び出し元に委ねます。
func Load(args []string, out io.Writer) (*Config, error) {
	cfg := DefaultConfig()
	fs := flag.NewFlagSet("codepack", flag.ContinueOnError)
	fs.SetOutput(out)

	// Note: cmd側で詳細なUsageを表示するため、ここではデフォルトの挙動のままにするか、
	// シンプルなエラーメッセージのみを出力するように設計します。

	fs.StringVar(&cfg.TargetDir, "d", cfg.TargetDir, "Target directory")
	fs.StringVar(&cfg.OutputFile, "o", cfg.OutputFile, "Output file")
	fs.BoolVar(&cfg.CopyToClipboard, "c", false, "Copy to clipboard")
	
	var patterns, ignores arrayFlags
	fs.Var(&patterns, "p", "Ignore patterns")
	fs.Var(&ignores, "i", "Ignore files")

	fs.StringVar(&cfg.LanguageMap, "m", "", "Language map JSON")
	fs.BoolVar(&cfg.ForceLarge, "force-large", false, "Force include large files")
	fs.BoolVar(&cfg.SkipLarge, "skip-large", false, "Skip large files")
	fs.BoolVar(&cfg.ShowVersion, "v", false, "Show version")
	fs.BoolVar(&cfg.ShowVersion, "version", false, "Show version")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg.IgnorePatterns = patterns
	cfg.IgnoreFiles = ignores

	if cfg.ForceLarge && cfg.SkipLarge {
		return nil, errors.New("--force-large and --skip-large cannot be used together")
	}

	return cfg, nil
}
