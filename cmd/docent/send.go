package main

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/relevatist/docent/internal/config"
	"github.com/relevatist/docent/internal/send"
	"github.com/relevatist/docent/internal/zellij"
)

const sendUsage = `usage: docent send ref <file>[:<start>[-<end>]]
       docent send text < input

Paste into the harness input of this tab. Never submits it.`

var errSendUsage = errors.New("usage: docent send ref <file>[:<start>[-<end>]] | text")

func runSend(args []string) int {
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help" || args[0] == "help") {
		if _, err := fmt.Println(sendUsage); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "docent send:", err)
			return 1
		}
		return 0
	}
	err := doSend(zellij.Exec{}, os.Stdin, args)
	if errors.Is(err, errSendUsage) {
		_, _ = fmt.Fprintln(os.Stderr, "docent send:", sendUsage)
		return 1
	}
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "docent send:", err)
		return 1
	}
	return 0
}

func doSend(r zellij.Runner, stdin io.Reader, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	switch {
	case len(args) == 2 && args[0] == "ref":
		h, _ := cfg.Resolve(cfg.Harness)
		return send.PasteRef(r, args[1], h.Kind, cfg.FocusOnSend)
	case len(args) == 1 && args[0] == "text":
		b, err := io.ReadAll(stdin)
		if err != nil {
			return err
		}
		return send.Paste(r, string(b), cfg.FocusOnSend)
	}
	return errSendUsage
}
