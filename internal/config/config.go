package config

// Config はアプリケーションの実行設定を保持します。
type Config struct {
	TargetDir       string
	OutputFile      string
	CopyToClipboard bool
	IgnorePatterns  []string // -p flags
	IgnoreFiles     []string // -i flags
	LanguageMap     string   // -m flag
	ForceLarge      bool     // --force-large
	SkipLarge       bool     // --skip-large
	ShowVersion     bool
}

// DefaultConfig はデフォルト設定を返します。
func DefaultConfig() *Config {
	return &Config{
		TargetDir:      ".",
		OutputFile:     "codebase.md",
		IgnorePatterns: []string{},
		IgnoreFiles:    []string{},
	}
}
