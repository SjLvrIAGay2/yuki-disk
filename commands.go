package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

func run(ctx context.Context, args []string) (result error) {
	flags := flag.NewFlagSet("yuki", flag.ContinueOnError)
	config := flags.String("config", "config.json", "strict JSON config")
	idem := flags.String("idempotency-key", "", "stable mutation key")
	if e := flags.Parse(args); e != nil {
		return e
	}
	args = flags.Args()
	if len(args) == 0 {
		return errors.New("commands: upload LOCAL PATH | download VERSION TARGET | ls [DIR] | mkdir PATH | mv/cp SOURCE TARGET_DIR | rename SOURCE NAME | versions PATH | trash/restore PATH... | version-trash ID... | version-restore/version-purge ID TRASH_ID | version-recycle | mounts | files | recycle")
	}
	b, e := os.ReadFile(*config)
	if e != nil {
		return e
	}
	var cfg Config
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if e = d.Decode(&cfg); e != nil {
		return e
	}
	if d.Decode(&struct{}{}) != io.EOF {
		return errors.New("trailing config data")
	}
	if !validURLPath(cfg.Control, cfg.Development, true) || len(cfg.Password) < 12 || len(cfg.Password) > 1024 {
		return errors.New("invalid config")
	}
	for _, n := range cfg.Nodes {
		if !validURL(n, cfg.Development) {
			return errors.New("invalid node URL")
		}
	}
	c := Client{cfg: cfg, http: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect forbidden") }}}
	if e = c.login(ctx); e != nil {
		return e
	}
	logs := c.startCommandLogs()
	closeLogs := logs.start()
	defer func() {
		if result != nil {
			logs.add("warn", "command_failed")
		} else {
			logs.add("info", "command_completed")
		}
		closeLogs()
	}()
	var out any
	switch args[0] {
	case "ls":
		if len(args) > 2 {
			return errors.New("ls [DIR]")
		}
		dir := ""
		if len(args) == 2 {
			dir = args[1]
		}
		e = c.request(ctx, "GET", "/v1/browse?path="+url.QueryEscape(dir), nil, "", &out)
	case "mkdir":
		if len(args) != 2 {
			return errors.New("mkdir PATH")
		}
		e = c.request(ctx, "POST", "/v1/folders", map[string]string{"path": args[1]}, "", &out)
	case "mv", "cp", "rename":
		if len(args) != 3 {
			return errors.New("mv/cp SOURCE TARGET_DIR | rename SOURCE NAME")
		}
		op := map[string]string{"mv": "move", "cp": "copy", "rename": "rename"}[args[0]]
		target := args[2]
		if op == "rename" {
			if strings.ContainsAny(target, "/\\") {
				return errors.New("new name must not contain separators")
			}
			target = path.Join(path.Dir(args[1]), target)
		}
		key := idempotencyKey(*idem)
		e = c.request(ctx, "POST", "/v1/operations", map[string]any{"operation": op, "paths": []string{args[1]}, "target": target}, key, &out)
	case "visibility":
		return errors.New("legacy visibility links are retired; create a scoped file share in web")
	case "mounts":
		e = c.request(ctx, "GET", "/v1/mounts", nil, "", &out)
	case "versions":
		if len(args) != 2 {
			return errors.New("versions PATH")
		}
		e = c.request(ctx, "GET", "/v1/versions?path="+url.QueryEscape(args[1]), nil, "", &out)
	case "version-trash", "version-restore", "version-purge", "version-recycle":
		e = c.versionCommand(ctx, args, *idem, &out)
	case "upload":
		if len(args) != 3 {
			return errors.New("upload LOCAL PATH")
		}
		id, e := c.upload(ctx, args[1], args[2], *idem)
		if e != nil {
			return e
		}
		out = map[string]string{"version_id": id}
	case "download":
		if len(args) != 3 {
			return errors.New("download VERSION TARGET")
		}
		if e = c.download(ctx, args[1], args[2]); e != nil {
			return e
		}
		out = map[string]string{"saved": args[2]}
	case "trash", "restore":
		if len(args) < 2 {
			return errors.New("path required")
		}
		key := idempotencyKey(*idem)
		e = c.request(ctx, "POST", "/v1/operations", map[string]any{"operation": args[0], "paths": args[1:]}, key, &out)
	case "files", "recycle":
		path := "/v1/" + args[0]
		if args[0] == "recycle" {
			path = "/v1/files?trash=true"
		}
		e = c.request(ctx, "GET", path, nil, "", &out)
	case "block", "events", "tasks":
		return errors.New("system diagnostics require the web administrator interface")
	default:
		return errors.New("unknown command " + strconv.Quote(args[0]))
	}
	if e != nil {
		return e
	}
	return json.NewEncoder(os.Stdout).Encode(out)
}
