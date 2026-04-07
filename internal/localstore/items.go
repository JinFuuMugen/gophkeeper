package localstore

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/JinFuuMugen/GophKeeper/internal/cliapp/clientapi"
)

func LoadLocalItems(dir string) ([]clientapi.Item, error) {
	b, err := os.ReadFile(itemsPath(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read items: %w", err)
	}
	var items []clientapi.Item
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, fmt.Errorf("parse items: %w", err)
	}
	return items, nil
}

func ReplaceLocalItems(dir string, items []clientapi.Item) error {
	b, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal items: %w", err)
	}
	if err := os.WriteFile(itemsPath(dir), b, 0o600); err != nil {
		return fmt.Errorf("write items: %w", err)
	}
	return nil
}

func UpsertLocalItem(dir string, it clientapi.Item) error {
	items, err := LoadLocalItems(dir)
	if err != nil {
		return err
	}

	for i := range items {
		if items[i].ID == it.ID {
			items[i] = it
			return ReplaceLocalItems(dir, items)
		}
	}
	items = append(items, it)
	return ReplaceLocalItems(dir, items)
}
