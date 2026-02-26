package cliapp

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/JinFuuMugen/GophKeeper/internal/cliapp/clientapi"
	"github.com/JinFuuMugen/GophKeeper/internal/cryptokit"
	"github.com/JinFuuMugen/GophKeeper/internal/localstore"
)

const helpMessage = `GophKeeper CLI (client-side encryption)

Usage:
  gophkeeper <command> [flags]

Commands:
  register     Register new user on server
  login        Login and save token + kdf_salt locally
  add-text     Add encrypted text item
  add-login    Add encrypted login/password item
  add-card     Add encrypted bank card item
  add-binary   Add encrypted binary item from file
  list         List items (from local cache or after sync)
  sync         Pull updates from server since last sync, update local cache
  get          Decrypt and show one item by id (needs master password)
  delete       Mark item deleted (creates tombstone with higher version)
  version      Print build info

Global env:
  GK_SERVER_URL   Server base URL
  GK_MASTER_PASS  Master password`

type BuildInfo struct {
	Version string
	Date    string
	Commit  string
}

type App struct {
	build BuildInfo
}

func New(build BuildInfo) *App {
	return &App{build: build}
}

func (a *App) Run(args []string) error {
	if len(args) == 0 {
		a.printHelp()
		return nil
	}

	cmd := args[0]
	switch cmd {
	case "help", "-h", "--help":
		a.printHelp()
		return nil
	case "version", "--version", "-version":
		fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", a.build.Version, a.build.Date, a.build.Commit)
		return nil
	case "register":
		return a.cmdRegister(args[1:])
	case "login":
		return a.cmdLogin(args[1:])
	case "add-text":
		return a.cmdAddText(args[1:])
	case "add-login":
		return a.cmdAddLogin(args[1:])
	case "add-card":
		return a.cmdAddCard(args[1:])
	case "add-binary":
		return a.cmdAddBinary(args[1:])
	case "list":
		return a.cmdList(args[1:])
	case "sync":
		return a.cmdSync(args[1:])
	case "get":
		return a.cmdGet(args[1:])
	case "delete":
		return a.cmdDelete(args[1:])
	default:
		return fmt.Errorf("unknown command %q (use: gophkeeper help)", cmd)
	}
}

func (a *App) printHelp() {
	fmt.Println(helpMessage)
}

func defaultServerURL() string {
	if v := os.Getenv("GK_SERVER_URL"); v != "" {
		return v
	}
	return "http://localhost:8080"
}

func masterPassFrom(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	if v := os.Getenv("GK_MASTER_PASS"); v != "" {
		return v, nil
	}
	return "", errors.New("master password required: use --master-pass or GK_MASTER_PASS env")
}

func appDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".gophkeeper"), nil
}

func (a *App) cmdRegister(args []string) error {
	fs := flag.NewFlagSet("register", flag.ContinueOnError)
	var serverURL, login, password string
	fs.StringVar(&serverURL, "server", defaultServerURL(), "server base url")
	fs.StringVar(&login, "login", "", "login")
	fs.StringVar(&password, "password", "", "password")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if login == "" || password == "" {
		return errors.New("--login and --password required")
	}

	api := clientapi.New(serverURL)
	id, err := api.Register(login, password)
	if err != nil {
		return err
	}
	fmt.Println("registered user_id:", id)
	return nil
}

func (a *App) cmdLogin(args []string) error {
	fs := flag.NewFlagSet("login", flag.ContinueOnError)
	var serverURL, login, password string
	fs.StringVar(&serverURL, "server", defaultServerURL(), "server base url")
	fs.StringVar(&login, "login", "", "login")
	fs.StringVar(&password, "password", "", "password")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if login == "" || password == "" {
		return errors.New("--login and --password required")
	}

	api := clientapi.New(serverURL)
	resp, err := api.Login(login, password)
	if err != nil {
		return err
	}

	if resp.KDFSaltB64 == "" {
		return errors.New("server returned empty kdf_salt_b64")
	}

	dir, err := appDir()
	if err != nil {
		return err
	}
	if err := localstore.EnsureDir(dir); err != nil {
		return err
	}

	cfg := localstore.Config{
		ServerURL:     serverURL,
		Token:         resp.Token,
		KDFSaltB64:    resp.KDFSaltB64,
		LastSyncRFC:   "",
		ConfigVersion: 1,
	}
	if err := localstore.SaveConfig(dir, cfg); err != nil {
		return err
	}

	fmt.Println("login ok: token and kdf_salt saved to", filepath.Join(dir, "config.json"))
	return nil
}

type payloadText struct {
	Text     string `json:"text"`
	Metadata string `json:"metadata,omitempty"`
}

type payloadLogin struct {
	Site     string `json:"site,omitempty"`
	Login    string `json:"login"`
	Password string `json:"password"`
	Metadata string `json:"metadata,omitempty"`
}

type payloadCard struct {
	Number   string `json:"number"`
	ExpMMYY  string `json:"exp_mmyy"`
	CVC      string `json:"cvc"`
	Holder   string `json:"holder,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

type payloadBinary struct {
	Filename string `json:"filename"`
	Data     []byte `json:"data"`
	Metadata string `json:"metadata,omitempty"`
}

func (a *App) loadSession() (dir string, cfg localstore.Config, api *clientapi.Client, err error) {
	dir, err = appDir()
	if err != nil {
		return "", localstore.Config{}, nil, err
	}
	cfg, err = localstore.LoadConfig(dir)
	if err != nil {
		return "", localstore.Config{}, nil, err
	}
	api = clientapi.New(cfg.ServerURL)
	api.SetToken(cfg.Token)
	return dir, cfg, api, nil
}

func (a *App) deriveCrypt(masterPass string, cfg localstore.Config) (*cryptokit.MasterCrypt, error) {
	salt, err := base64.StdEncoding.DecodeString(cfg.KDFSaltB64)
	if err != nil {
		return nil, fmt.Errorf("decode kdf salt: %w", err)
	}
	key := cryptokit.DeriveKeyArgon2id([]byte(masterPass), salt, cryptokit.DefaultKDFParams())
	return cryptokit.New(key)
}

func (a *App) cmdAddText(args []string) error {
	fs := flag.NewFlagSet("add-text", flag.ContinueOnError)
	var title, text, meta, mp string
	fs.StringVar(&title, "title", "", "non-sensitive title (stored in items.metadata column)")
	fs.StringVar(&text, "text", "", "text to store (sensitive)")
	fs.StringVar(&meta, "meta", "", "sensitive metadata (stored inside ciphertext)")
	fs.StringVar(&mp, "master-pass", "", "master password (or GK_MASTER_PASS env)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if title == "" || text == "" {
		return errors.New("--title and --text required")
	}

	mp, err := masterPassFrom(mp)
	if err != nil {
		return err
	}

	dir, cfg, api, err := a.loadSession()
	if err != nil {
		return err
	}
	crypt, err := a.deriveCrypt(mp, cfg)
	if err != nil {
		return err
	}

	body, _ := json.Marshal(payloadText{Text: text, Metadata: meta})
	enc, err := crypt.Encrypt(body, []byte("text"))
	if err != nil {
		return err
	}

	item, err := api.UpsertItem(clientapi.UpsertItemRequest{
		Type:         "text",
		EncryptedB64: base64.StdEncoding.EncodeToString(enc),
		Metadata:     title,
		Version:      1,
		Deleted:      false,
	})
	if err != nil {
		return err
	}

	_ = localstore.UpsertLocalItem(dir, item)
	fmt.Println("saved item id:", item.ID)
	return nil
}

func (a *App) cmdAddLogin(args []string) error {
	fs := flag.NewFlagSet("add-login", flag.ContinueOnError)
	var title, site, login, password, meta, mp string
	fs.StringVar(&title, "title", "", "non-sensitive title")
	fs.StringVar(&site, "site", "", "site (sensitive, inside ciphertext)")
	fs.StringVar(&login, "login", "", "login (sensitive)")
	fs.StringVar(&password, "password", "", "password (sensitive)")
	fs.StringVar(&meta, "meta", "", "sensitive metadata")
	fs.StringVar(&mp, "master-pass", "", "master password (or GK_MASTER_PASS env)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if title == "" || login == "" || password == "" {
		return errors.New("--title, --login, --password required")
	}

	mp, err := masterPassFrom(mp)
	if err != nil {
		return err
	}
	dir, cfg, api, err := a.loadSession()
	if err != nil {
		return err
	}
	crypt, err := a.deriveCrypt(mp, cfg)
	if err != nil {
		return err
	}

	body, _ := json.Marshal(payloadLogin{Site: site, Login: login, Password: password, Metadata: meta})
	enc, err := crypt.Encrypt(body, []byte("login"))
	if err != nil {
		return err
	}

	item, err := api.UpsertItem(clientapi.UpsertItemRequest{
		Type:         "login",
		EncryptedB64: base64.StdEncoding.EncodeToString(enc),
		Metadata:     title,
		Version:      1,
		Deleted:      false,
	})
	if err != nil {
		return err
	}

	_ = localstore.UpsertLocalItem(dir, item)
	fmt.Println("saved item id:", item.ID)
	return nil
}

func (a *App) cmdAddCard(args []string) error {
	fs := flag.NewFlagSet("add-card", flag.ContinueOnError)
	var title, number, exp, cvc, holder, meta, mp string
	fs.StringVar(&title, "title", "", "non-sensitive title")
	fs.StringVar(&number, "number", "", "card number (sensitive)")
	fs.StringVar(&exp, "exp", "", "exp MMYY (sensitive)")
	fs.StringVar(&cvc, "cvc", "", "cvc (sensitive)")
	fs.StringVar(&holder, "holder", "", "holder name (sensitive)")
	fs.StringVar(&meta, "meta", "", "sensitive metadata")
	fs.StringVar(&mp, "master-pass", "", "master password (or GK_MASTER_PASS env)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if title == "" || number == "" || exp == "" || cvc == "" {
		return errors.New("--title, --number, --exp, --cvc required")
	}

	mp, err := masterPassFrom(mp)
	if err != nil {
		return err
	}
	dir, cfg, api, err := a.loadSession()
	if err != nil {
		return err
	}
	crypt, err := a.deriveCrypt(mp, cfg)
	if err != nil {
		return err
	}

	body, _ := json.Marshal(payloadCard{Number: number, ExpMMYY: exp, CVC: cvc, Holder: holder, Metadata: meta})
	enc, err := crypt.Encrypt(body, []byte("card"))
	if err != nil {
		return err
	}

	item, err := api.UpsertItem(clientapi.UpsertItemRequest{
		Type:         "card",
		EncryptedB64: base64.StdEncoding.EncodeToString(enc),
		Metadata:     title,
		Version:      1,
		Deleted:      false,
	})
	if err != nil {
		return err
	}

	_ = localstore.UpsertLocalItem(dir, item)
	fmt.Println("saved item id:", item.ID)
	return nil
}

func (a *App) cmdAddBinary(args []string) error {
	fs := flag.NewFlagSet("add-binary", flag.ContinueOnError)
	var title, path, meta, mp string
	fs.StringVar(&title, "title", "", "non-sensitive title")
	fs.StringVar(&path, "path", "", "file path")
	fs.StringVar(&meta, "meta", "", "sensitive metadata")
	fs.StringVar(&mp, "master-pass", "", "master password (or GK_MASTER_PASS env)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if title == "" || path == "" {
		return errors.New("--title and --path required")
	}

	mp, err := masterPassFrom(mp)
	if err != nil {
		return err
	}
	dir, cfg, api, err := a.loadSession()
	if err != nil {
		return err
	}
	crypt, err := a.deriveCrypt(mp, cfg)
	if err != nil {
		return err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	body, _ := json.Marshal(payloadBinary{
		Filename: filepath.Base(path),
		Data:     data,
		Metadata: meta,
	})
	enc, err := crypt.Encrypt(body, []byte("binary"))
	if err != nil {
		return err
	}

	item, err := api.UpsertItem(clientapi.UpsertItemRequest{
		Type:         "binary",
		EncryptedB64: base64.StdEncoding.EncodeToString(enc),
		Metadata:     title,
		Version:      1,
		Deleted:      false,
	})
	if err != nil {
		return err
	}

	_ = localstore.UpsertLocalItem(dir, item)
	fmt.Println("saved item id:", item.ID)
	return nil
}

func (a *App) cmdList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	var fromServer bool
	fs.BoolVar(&fromServer, "server", false, "fetch from server (otherwise use local cache)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dir, cfg, api, err := a.loadSession()
	if err != nil {
		return err
	}

	var items []clientapi.Item
	if fromServer {
		items, err = api.ListItems()
		if err != nil {
			return err
		}
		_ = localstore.ReplaceLocalItems(dir, items)
	} else {
		items, _ = localstore.LoadLocalItems(dir)
		if len(items) == 0 {
			fmt.Println("local cache empty; run: gophkeeper sync")
			return nil
		}
		_ = cfg // silence
	}

	for _, it := range items {
		del := ""
		if it.Deleted {
			del = " (deleted)"
		}
		fmt.Printf("%s  type=%s  title=%q  ver=%d  updated=%s%s\n",
			it.ID, it.Type, it.Metadata, it.Version, it.UpdatedAt.Format(time.RFC3339), del)
	}
	return nil
}

func (a *App) cmdSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	var full bool
	fs.BoolVar(&full, "full", false, "full sync (ignore since)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	dir, cfg, api, err := a.loadSession()
	if err != nil {
		return err
	}

	var items []clientapi.Item
	if full || cfg.LastSyncRFC == "" {
		items, err = api.ListItems()
		if err != nil {
			return err
		}
	} else {
		items, err = api.SyncItems(cfg.LastSyncRFC)
		if err != nil {
			return err
		}
	}

	// merge into local cache
	if full || cfg.LastSyncRFC == "" {
		if err := localstore.ReplaceLocalItems(dir, items); err != nil {
			return err
		}
	} else {
		for _, it := range items {
			_ = localstore.UpsertLocalItem(dir, it)
		}
	}

	cfg.LastSyncRFC = time.Now().UTC().Format(time.RFC3339Nano)
	if err := localstore.SaveConfig(dir, cfg); err != nil {
		return err
	}

	fmt.Println("sync ok; received:", len(items), "items; lastSync set to", cfg.LastSyncRFC)
	return nil
}

func (a *App) cmdGet(args []string) error {
	fs := flag.NewFlagSet("get", flag.ContinueOnError)
	var idStr, mp string
	fs.StringVar(&idStr, "id", "", "item id (uuid)")
	fs.StringVar(&mp, "master-pass", "", "master password (or GK_MASTER_PASS env)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if idStr == "" {
		return errors.New("--id required")
	}

	mp, err := masterPassFrom(mp)
	if err != nil {
		return err
	}

	dir, cfg, _, err := a.loadSession()
	if err != nil {
		return err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("bad uuid: %w", err)
	}

	items, err := localstore.LoadLocalItems(dir)
	if err != nil {
		return err
	}
	var found *clientapi.Item
	for i := range items {
		if items[i].ID == id.String() {
			found = &items[i]
			break
		}
	}
	if found == nil {
		return errors.New("item not found in local cache; run: gophkeeper sync")
	}
	if found.Deleted {
		return errors.New("item is deleted")
	}
	if found.EncryptedB64 == "" {
		return errors.New("missing encrypted payload in cache")
	}

	crypt, err := a.deriveCrypt(mp, cfg)
	if err != nil {
		return err
	}

	raw, err := base64.StdEncoding.DecodeString(found.EncryptedB64)
	if err != nil {
		return fmt.Errorf("decode encrypted_b64: %w", err)
	}

	plain, err := crypt.Decrypt(raw, []byte(found.Type))
	if err != nil {
		return fmt.Errorf("decrypt failed (wrong master password?): %w", err)
	}

	fmt.Printf("id: %s\n", found.ID)
	fmt.Printf("type: %s\n", found.Type)
	fmt.Printf("title: %s\n", found.Metadata)
	fmt.Printf("version: %d\n", found.Version)
	fmt.Printf("updated: %s\n", found.UpdatedAt.Format(time.RFC3339))
	fmt.Println("payload:")

	// pretty print json if possible
	var anyJSON any
	if json.Unmarshal(plain, &anyJSON) == nil {
		b, _ := json.MarshalIndent(anyJSON, "", "  ")
		fmt.Println(string(b))
	} else {
		fmt.Println(string(plain))
	}

	return nil
}

func (a *App) cmdDelete(args []string) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	var idStr string
	fs.StringVar(&idStr, "id", "", "item id (uuid)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if idStr == "" {
		return errors.New("--id required")
	}

	dir, _, api, err := a.loadSession()
	if err != nil {
		return err
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("bad uuid: %w", err)
	}

	// determine next version from local cache (simple approach)
	items, _ := localstore.LoadLocalItems(dir)
	var curVer int64 = 0
	var itType string
	var title string
	for i := range items {
		if items[i].ID == id.String() {
			curVer = items[i].Version
			itType = items[i].Type
			title = items[i].Metadata
			break
		}
	}
	if itType == "" {
		// still can delete without knowing type/title, but keep type empty is invalid for server
		return errors.New("unknown item type in local cache; run sync before delete")
	}

	ver := curVer + 1
	out, err := api.UpsertItem(clientapi.UpsertItemRequest{
		ID:      id.String(),
		Type:    itType,
		Version: ver,
		Deleted: true,
		// EncryptedB64 omitted for deleted
		Metadata: title,
	})
	if err != nil {
		return err
	}
	_ = localstore.UpsertLocalItem(dir, out)
	fmt.Println("deleted (tombstone saved) id:", out.ID, "version:", out.Version)
	return nil
}

// small helper for future improvements
func normalizeServerURL(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "/")
	return s
}
