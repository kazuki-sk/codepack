package ignorer

import (
	"bufio"
	"io"
	"path"
	"path/filepath"
	"strings"
)

// Matcher はパスが除外対象か判定するインターフェースです。
// .gitignore仕様に対応するため isDir 引数を追加しました。
type Matcher interface {
	Match(path string, isDir bool) bool
}

// ignoreRule は1行分のルールを表します。
type ignoreRule struct {
	pattern string
	negate  bool // '!' で始まる場合
	dirOnly bool // '/' で終わる場合
}

// GitIgnoreMatcher は.gitignore仕様に近いルールマッチングを提供します。
// 定義順に評価し、後方のルールが前方の判定を上書きする（後勝ち）ことで、
// 「否定(!)」ルールを含む最終的な判定を行います。
type GitIgnoreMatcher struct {
	rules []ignoreRule
}

// NewGitIgnoreMatcher はReaderからルールを読み込みます。
func NewGitIgnoreMatcher(r io.Reader) *GitIgnoreMatcher {
	m := &GitIgnoreMatcher{}
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 1. コメント(#) と 空行のスキップ
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		rule := ignoreRule{
			pattern: line,
			negate:  false,
			dirOnly: false,
		}

		// 2. 否定(!) の処理
		if strings.HasPrefix(rule.pattern, "!") {
			rule.negate = true
			rule.pattern = strings.TrimPrefix(rule.pattern, "!")
		}

		// 3. 末尾スラッシュ(ディレクトリ限定) の処理
		if strings.HasSuffix(rule.pattern, "/") {
			rule.dirOnly = true
			rule.pattern = strings.TrimSuffix(rule.pattern, "/")
		}

		m.rules = append(m.rules, rule)
	}
	return m
}

// Match はパスがルールセットにより除外されるかを判定します。
// gitignoreの仕様に従い、リストの下にあるルールが優先されます。
// ここでは簡易的に、マッチした最後のルールの結果（除外か、包含か）を返します。
func (m *GitIgnoreMatcher) Match(targetPath string, isDir bool) bool {
	ignored := false

	// パス区切り文字の正規化（全てスラッシュ '/' に変換）
	targetPath = filepath.ToSlash(targetPath)
	fileName := filepath.Base(targetPath)

	for _, rule := range m.rules {
		// ディレクトリ限定ルールで、対象がディレクトリでない場合はスキップ
		if rule.dirOnly && !isDir {
			continue
		}

		matched := false

		// A. 単純なファイル名マッチ (ディレクトリを含まない場合)
		if !strings.Contains(rule.pattern, "/") {
			// レビュー対応: クロスプラットフォーム対応のため filepath.Match ではなく path.Match を使用
			if ok, _ := path.Match(rule.pattern, fileName); ok {
				matched = true
			}
		} else {
			// B. パスを含むマッチ (簡易実装)
			// 注意: 本格的な `**` 対応やルート指定(`/`)の完全実装は標準ライブラリのみでは複雑なため、
			// Phase 2では `path.Match` と `strings.Contains` を組み合わせた近似実装とします。

			// 先頭の `/` を削除して相対パス比較をしやすくする
			cleanPattern := strings.TrimPrefix(rule.pattern, "/")

			// 単純比較またはグロブマッチ
			if targetPath == cleanPattern || strings.HasSuffix(targetPath, "/"+cleanPattern) {
				matched = true
			} else if ok, _ := path.Match(cleanPattern, targetPath); ok {
				matched = true
			}
		}

		if matched {
			if rule.negate {
				ignored = false // 除外しない（含める）
			} else {
				ignored = true // 除外する
			}
		}
	}

	return ignored
}
