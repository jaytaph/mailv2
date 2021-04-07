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

package bitmaelumClient

import (
	"github.com/bitmaelum/bitmaelum-suite/internal/config"
	"github.com/bitmaelum/bitmaelum-suite/internal/container"
	"github.com/bitmaelum/bitmaelum-suite/internal/vault"
	"github.com/bitmaelum/bitmaelum-suite/pkg/address"
	"github.com/bitmaelum/bitmaelum-suite/pkg/bmcrypto"
	"github.com/pkg/errors"
)

const defaultResolver = "https://resolver.bitmaelum.com"

type client struct {
	Address    *address.Address
	Name       string
	PrivateKey *bmcrypto.PrivKey
	Vault      *vault.Vault
}

type BitMaelumClient struct {
	client      client
	resolverURL string
}

func NewBitMaelumClient() *BitMaelumClient {
	return &BitMaelumClient{
		resolverURL: defaultResolver,
	}
}

func (b *BitMaelumClient) SetResolver(url string) {
	b.resolverURL = url
}

func (b *BitMaelumClient) SetClientFromVault(accountAddress string) error {
	if b.client.Vault == nil {
		return errors.Errorf("vault not loaded")
	}

	for _, acc := range b.client.Vault.Store.Accounts {
		if acc.Address.String() == accountAddress {
			b.client.Address = acc.Address
			b.client.Name = acc.Name
			*b.client.PrivateKey = acc.GetActiveKey().PrivKey
			return nil
		}
	}

	return errors.Errorf("account %s not found on vault", accountAddress)
}

func (b *BitMaelumClient) SetClientFromMnemonic(accountAddress, name, mnemonic string) error {
	err := b.parseAccountAndName(accountAddress, name)
	if err != nil {
		return err
	}

	// Now generate a new key from the mnemonic
	kp, err := bmcrypto.GenerateKeypairFromMnemonic(mnemonic)
	if err != nil {
		return errors.Wrap(err, "parsing mnemonic")
	}
	*b.client.PrivateKey = kp.PrivKey

	return nil
}

func (b *BitMaelumClient) SetClientFromPrivateKey(accountAddress, name, privKey string) error {
	err := b.parseAccountAndName(accountAddress, name)
	if err != nil {
		return err
	}

	// Convert privKey string to bmcrypto
	b.client.PrivateKey, err = bmcrypto.PrivateKeyFromString(privKey)
	if err != nil {
		return errors.Wrap(err, "parsing private key")
	}

	return nil
}

func (b *BitMaelumClient) parseAccountAndName(accountAddress, name string) error {
	var err error

	config.Client.Resolver.Remote.Enabled = true
	config.Client.Resolver.Remote.URL = b.resolverURL

	b.client.Address, err = address.NewAddress(accountAddress)
	if err != nil {
		return errors.Wrap(err, "parsing account address")
	}

	// Verify client exists
	svc := container.Instance.GetResolveService()
	_, err = svc.ResolveAddress(b.client.Address.Hash())
	if err != nil {
		return errors.Wrap(err, "resolving client address")
	}

	b.client.Name = name

	return nil
}
