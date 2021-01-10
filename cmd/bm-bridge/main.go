// Copyright (c) 2021 BitMaelum Authors
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	common "github.com/bitmaelum/bitmaelum-suite/cmd/bm-bridge/internal"

	imapgw "github.com/bitmaelum/bitmaelum-suite/cmd/bm-bridge/internal/imap/backend"
	smtpgw "github.com/bitmaelum/bitmaelum-suite/cmd/bm-bridge/internal/smtp/backend"
	bminternal "github.com/bitmaelum/bitmaelum-suite/internal"
	"github.com/bitmaelum/bitmaelum-suite/internal/config"
	"github.com/bitmaelum/bitmaelum-suite/internal/vault"
	"github.com/emersion/go-imap/server"
	"github.com/emersion/go-smtp"
	"github.com/sirupsen/logrus"
)

type options struct {
	ImapHost       string `long:"imaphost" description:"Host:Port to imap server from" required:"false"`
	SMTPHost       string `long:"smtphost" description:"Host:Port to smtp server from" required:"false"`
	GatewayAccount string `long:"gatewayaccount" description:"Account to be used for a email->bitmaelum gateway. If empty it will require authentication to send mails." default:""`
	Config         string `short:"c" long:"config" description:"Path to your configuration file"`
	Password       string `short:"p" long:"password" description:"Vault password" default:""`
	Vault          string `long:"vault" description:"Custom vault file" default:""`
	Version        bool   `short:"v" long:"version" description:"Display version information"`
	Debug          bool   `long:"debug" description:"It will print all the communications to this imap/smtp server"`
}

// Opts are the options set through the command line
var opts options

func main() {
	rand.Seed(time.Now().UnixNano())

	bminternal.ParseOptions(&opts)
	if opts.Version {
		bminternal.WriteVersionInfo("BitMaelum IMAP", os.Stdout)
		fmt.Println()
		os.Exit(0)
	}

	if opts.ImapHost == "" && opts.SMTPHost == "" {
		fmt.Println(" either imaphost or smtphost needs to be specified")
		os.Exit(0)
	}

	config.LoadClientConfig(opts.Config)

	// Set default vault info if set in config
	vault.VaultPassword = opts.Password
	vault.VaultPath = config.Client.Vault.Path
	if opts.Vault != "" {
		vault.VaultPath = opts.Vault
	}

	v := vault.OpenDefaultVault()

	fmt.Println(" * BM-BRIDGE *")

	// setup context so we can easily stop all components of the server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wait for signals and cancel context
	go setupSignals(cancel)

	// Start the SMTP server
	go startSMTPServer(v, cancel)

	if opts.GatewayAccount != "" {
		// If gateway mode then start to fetch for pending mails
		go startFetcher(ctx, cancel, v, opts.GatewayAccount)
	} else {
		// If local mode then start the imap server too
		go startImapServer(v, cancel)
	}

	<-ctx.Done()

}

func setupSignals(cancel context.CancelFunc) {
	go func() {
		// Capture INT and TERM signals
		sigChannel := make(chan os.Signal, 1)
		signal.Notify(sigChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP)

		for {
			s := <-sigChannel
			fmt.Printf("Signal %s received. Terminating.\n", s)
			cancel()
		}
	}()
}

func startImapServer(v *vault.Vault, cancel context.CancelFunc) {
	be := imapgw.New(v)
	s := server.New(be)
	s.Addr = opts.ImapHost
	s.AllowInsecureAuth = true // We should run on TLS
	if opts.Debug {
		s.Debug = os.Stdout
	}

	fmt.Println(" [*] IMAP starting")

	if err := s.ListenAndServe(); err != nil {
		log.Fatal("error while accepting connection: " + err.Error() + "\n")
		cancel()
	}
}

func startSMTPServer(v *vault.Vault, cancel context.CancelFunc) {
	be := smtpgw.New(v, opts.GatewayAccount)
	s := smtp.NewServer(be)
	s.Addr = opts.SMTPHost
	s.Domain = "bitmaelum.network"
	s.ReadTimeout = 10 * time.Second
	s.WriteTimeout = 10 * time.Second
	s.MaxMessageBytes = 1024 * 1024
	s.MaxRecipients = 1
	s.AllowInsecureAuth = true
	if opts.Debug {
		s.Debug = os.Stdout
	}

	fmt.Print(" [*] SMTP starting ")
	if opts.GatewayAccount != "" {
		fmt.Println("(GW mode)")
	} else {
		fmt.Println("(Local mode)")
	}

	if err := s.ListenAndServe(); err != nil {
		log.Fatal("error while accepting connection: " + err.Error() + "\n")
		cancel()
	}
}

func startFetcher(ctx context.Context, cancel context.CancelFunc, v *vault.Vault, account string) {
	// @TODO we should use an API so we will receive a notification when a new
	// mail is delivered instead of pulling the account
	fetchTicker := time.NewTicker(30 * time.Second)

	fetcher := &common.Fetcher{
		Vault:   v,
		Account: account,
	}

	var err error
	fetcher.Info, fetcher.Client, err = common.GetClientAndInfo(v, account)
	if err != nil {
		log.Fatal("error while getting client/info\n")
		cancel()
	}

	for {
		select {
		// Process tickers
		case <-fetchTicker.C:
			go fetcher.CheckMail()

		// Context is done (signal send)
		case <-ctx.Done():
			logrus.Info("Stopping fetcher")
			return
		}
	}
}