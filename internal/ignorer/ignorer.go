package ignorer

import (
	"embed" // 追加
	"os"
)

// デフォルト設定を埋め込む
//go:embed default_ignore
var defaultIgnoreFS embed.FS

type Ignorer struct {
	matchers []Matcher
}

func NewIgnorer(matchers ...Matcher) *Ignorer {
	return &Ignorer{
		matchers: matchers,
	}
}

// デフォルトの除外ルールをロードするメソッド
func (i *Ignorer) LoadDefaults() error {
	f, err := defaultIgnoreFS.Open("default_ignore")
	if err != nil {
		return err
	}
	defer f.Close()

	// GitIgnoreMatcherを再利用してルールを追加
	matcher := NewGitIgnoreMatcher(f)
	// デフォルトルールは「最優先」ではなく「ベース」なので、リストの先頭に追加したいが、
	// 構造上は NewIgnorer 直後に呼べば先頭になるため、単純に AddMatcher でOK。
	i.AddMatcher(matcher)
	return nil
}

func (i *Ignorer) AddMatcher(m Matcher) {
	i.matchers = append(i.matchers, m)
}

func (i *Ignorer) ShouldIgnore(path string, isDir bool) bool {
	for _, m := range i.matchers {
		if m.Match(path, isDir) {
			return true
		}
	}
	return false
}

func (i *Ignorer) LoadIgnoreFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	matcher := NewGitIgnoreMatcher(f)
	i.AddMatcher(matcher)
	return nil
}
