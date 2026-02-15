package cmd

import (
	"encoding/json"
	"fmt"
)

// printJSON: 値をJSON整形して標準出力に出力する
func printJSON(v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}
