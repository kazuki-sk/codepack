package language

import (
	"embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

//go:embed extension_to_language.json
var defaultMapFS embed.FS

// Mapper は拡張子と言語名の対応関係を管理します。
type Mapper struct {
	// 仕様変更: LinguistMap形式 (map[string][]string) に対応
	// Key: 拡張子 (例: ".go"), Value: [言語名, 親言語...]
	extMap map[string][]string
}

// NewMapper はデフォルト設定とオプションのカスタム設定ファイルをロードしてMapperを作成します。
func NewMapper(customMapPath string) (*Mapper, error) {
	m := &Mapper{
		extMap: make(map[string][]string),
	}

	// 1. Load Defaults
	defaultData, err := defaultMapFS.ReadFile("extension_to_language.json")
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(defaultData, &m.extMap); err != nil {
		return nil, err
	}

	// 2. Merge Custom Config (if provided)
	if customMapPath != "" {
		customData, err := os.ReadFile(customMapPath)
		if err != nil {
			return nil, err
		}
		
		var customMap map[string][]string
		if err := json.Unmarshal(customData, &customMap); err != nil {
			return nil, err
		}

		// 上書きマージ
		for k, v := range customMap {
			m.extMap[k] = v
		}
	}

	return m, nil
}

// GetLanguage はファイルパス（拡張子）から言語名を返します。
// LinguistMap形式の配列の最初の要素を言語名として返します。
func (m *Mapper) GetLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if langs, ok := m.extMap[ext]; ok && len(langs) > 0 {
		return langs[0]
	}
	return ""
}
