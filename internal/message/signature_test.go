// Copyright (c) 2020 BitMaelum Authors
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

package message

import (
	"testing"

	"github.com/bitmaelum/bitmaelum-suite/internal/config"
	"github.com/bitmaelum/bitmaelum-suite/internal/container"
	"github.com/bitmaelum/bitmaelum-suite/internal/resolver"
	testing2 "github.com/bitmaelum/bitmaelum-suite/internal/testing"
	"github.com/bitmaelum/bitmaelum-suite/pkg/address"
	"github.com/bitmaelum/bitmaelum-suite/pkg/bmcrypto"
	"github.com/bitmaelum/bitmaelum-suite/pkg/proofofwork"
	"github.com/stretchr/testify/assert"
)

const (
	expectedServerSignature = "smq7qWrkwTL1vquhEAn/WoRDZBT1BjUQaUSsmTSSPePLRpM1sjX10mJwxXlivYOIlgTtJ+0SIeBM9rDPBSTw5JJhOS9ZFmpAYEPG9tkU+9EjxfBNnEBPrsYaqE81tO7OtY4xFrlYecdhepGUQSxbZQU4+Ih5jE9jLb+SuUxR6lGw2u2P+Ngy75dD33zlMTTgnmaVTxBueRlfArDExW5QE+pFv9/uFi8xM5a7eGnQHSjufQ9gM6WOWzhyWAnKI+6XMwx3MoQ53H3OU2vn4tPSUQQyxB+L4WTH9JtC0nLC0ggzvo5LOdCw4rCljsciYiEZ2WssGD9kXLGFIU/ixEX2Kw=="
	expectedClientSignature = "lIqI1QYBRHl7yRW367Lx2n/PFadrYDZ2a2NGSaL40EKum0ncOIXs8CIqKZ+LCUgmK2a9iH2d3mbXVPwZ3PBGsVgReaomyG6NrDbZ0PCbgnjmrmkVAFV0bDHlOxUl/BzyV+seIL7FL0lu+cODaHkmzH16FsZ5Vqcf1/Qe2GR/0Ka6xbWcIcajGsKtTx+WtGeZGZ5oLbAFatEjiv5gMAn2umKpP+w7uKhPa6CsYkv2YMVw+z/1NU2CO0jE6/2muihF9x4nPw6yiy+sXP86B26FQXLBcMgTZ4TAtzr/b2KvcEDj8y8HISs/YHJvTdqAXzYTPnha37ZIIZ7ce27Z41GAUQ=="
)

func TestSignHeader(t *testing.T) {
	setupServer()

	header := &Header{}
	_ = testing2.ReadJSON("../../testdata/header-002.json", &header)
	assert.Empty(t, header.Signatures.Server)
	err := SignServerHeader(header)
	assert.NoError(t, err)

	assert.Equal(t, expectedServerSignature, header.Signatures.Server)

	// Already present, don't overwrite
	_ = testing2.ReadJSON("../../testdata/header-002.json", &header)
	assert.NotEmpty(t, header.Signatures.Server)
	header.Signatures.Server = "foobar"
	err = SignServerHeader(header)
	assert.NoError(t, err)

	assert.Equal(t, "foobar", header.Signatures.Server)
}

func TestVerifyHeader(t *testing.T) {
	setupServer()

	header := &Header{}
	_ = testing2.ReadJSON("../../testdata/header-002.json", &header)
	assert.Empty(t, header.Signatures.Server)
	err := SignServerHeader(header)
	assert.NoError(t, err)
	assert.Equal(t, expectedServerSignature, header.Signatures.Server)

	// All is ok
	ok := VerifyServerHeader(*header)
	assert.True(t, ok)

	// Incorrect decoding
	header.Signatures.Server = "A"
	ok = VerifyServerHeader(*header)
	assert.False(t, ok)

	// Empty sig is not ok
	header.Signatures.Server = ""
	ok = VerifyServerHeader(*header)
	assert.False(t, ok)

	// incorrect key
	header.Signatures.Server = "Zm9vYmFy"
	ok = VerifyServerHeader(*header)
	assert.False(t, ok)
}

func setupServer() {
	addr, _ := address.NewAddress("foobar!")

	// Note: our mail server uses key1
	privKey, pubKey, err := testing2.ReadTestKey("../../testdata/key-1.json")
	if err != nil {
		panic(err)
	}
	config.Routing = config.RoutingConfig{
		RoutingID:  "12345678",
		PrivateKey: *privKey,
		PublicKey:  *pubKey,
	}

	// Setup container with mock repository for routing
	repo, _ := resolver.NewMockRepository()
	container.Instance.SetShared("resolver", func() (interface{}, error) {
		return resolver.KeyRetrievalService(repo), nil
	})

	uploadAddress(repo, *addr, "111100000000000000000000000097026f0daeaec1aeb8351b096637679cf350", "87654321", "../../testdata/key-2.json")
	uploadAddress(repo, *addr, "111100000000000000018f66a0f3591a883f2b9cc3e95a497e7cf9da1071b4cc", "12345678", "../../testdata/key-3.json")

	// Note: our mail server uses key1
	privKey, pubKey, err = testing2.ReadTestKey("../../testdata/key-1.json")
	if err != nil {
		panic(err)
	}
	ri := resolver.RoutingInfo{
		Hash:      "12345678",
		PublicKey: *pubKey,
		Routing:   "127.0.0.1",
	}
	_ = repo.UploadRouting(&ri, *privKey)
}

func uploadAddress(repo resolver.AddressRepository, addr address.Address, addrHash string, routingId string, keyPath string) {
	pow := proofofwork.NewWithoutProof(1, "foobar")

	privKey, pubKey, err := testing2.ReadTestKey(keyPath)
	if err != nil {
		panic(err)
	}

	ai := resolver.AddressInfo{
		Hash:        addrHash,
		PublicKey:   *pubKey,
		RoutingID:   routingId,
		Pow:         pow.String(),
		RoutingInfo: resolver.RoutingInfo{},
	}
	_ = repo.UploadAddress(addr, &ai, *privKey, *pow, "")

}

func TestSignClientHeader(t *testing.T) {
	privKey := setupClient()

	header := &Header{}
	_ = testing2.ReadJSON("../../testdata/header-001.json", &header)
	assert.Empty(t, header.Signatures.Client)
	err := SignClientHeader(header, *privKey)
	assert.NoError(t, err)
	assert.Equal(t, expectedClientSignature, header.Signatures.Client)

	// Already present, don't overwrite
	_ = testing2.ReadJSON("../../testdata/header-001.json", &header)
	assert.NotEmpty(t, header.Signatures.Client)
	header.Signatures.Client = "foobar"
	err = SignClientHeader(header, *privKey)
	assert.NoError(t, err)

	assert.Equal(t, "foobar", header.Signatures.Client)
}

func TestVerifyClientHeader(t *testing.T) {
	privKey := setupClient()

	header := &Header{}
	_ = testing2.ReadJSON("../../testdata/header-001.json", &header)
	assert.Empty(t, header.Signatures.Client)
	err := SignClientHeader(header, *privKey)
	assert.NoError(t, err)
	assert.Equal(t, expectedClientSignature, header.Signatures.Client)

	// All is ok
	ok := VerifyClientHeader(*header)
	assert.True(t, ok)

	// Incorrect decoding
	header.Signatures.Client = "A"
	ok = VerifyClientHeader(*header)
	assert.False(t, ok)

	// Empty sig is not ok
	header.Signatures.Client = ""
	ok = VerifyClientHeader(*header)
	assert.False(t, ok)

	// incorrect key
	header.Signatures.Client = "Zm9vYmFy"
	ok = VerifyClientHeader(*header)
	assert.False(t, ok)
}

func setupClient() *bmcrypto.PrivKey {
	addr, _ := address.NewAddress("foobar!")

	// Setup container with mock repository for routing
	repo, _ := resolver.NewMockRepository()
	container.Instance.SetShared("resolver", func() (interface{}, error) {
		return resolver.KeyRetrievalService(repo), nil
	})

	pow := proofofwork.NewWithoutProof(1, "foobar")
	var (
		ai resolver.AddressInfo
		ri resolver.RoutingInfo
	)

	privKey, pubKey, err := testing2.ReadTestKey("../../testdata/key-1.json")
	if err != nil {
		panic(err)
	}
	ai = resolver.AddressInfo{
		RoutingID:   "87654321",
		PublicKey:   *pubKey,
		RoutingInfo: resolver.RoutingInfo{},
		Pow:         pow.String(),
		Hash:        "000000000000000000000000000097026f0daeaec1aeb8351b096637679cf350",
	}
	_ = repo.UploadAddress(*addr, &ai, *privKey, *pow, "")

	ri = resolver.RoutingInfo{
		PublicKey: *pubKey,
		Routing:   "127.0.0.1",
		Hash:      "12345678",
	}

	_ = repo.UploadRouting(&ri, *privKey)

	// Note: our sender uses key3
	privKey, pubKey, err = testing2.ReadTestKey("../../testdata/key-3.json")
	if err != nil {
		panic(err)
	}

	ai = resolver.AddressInfo{
		RoutingID:   "12345678",
		RoutingInfo: resolver.RoutingInfo{},
		PublicKey:   *pubKey,
		Hash:        "000000000000000000018f66a0f3591a883f2b9cc3e95a497e7cf9da1071b4cc",
		Pow:         pow.String(),
	}
	_ = repo.UploadAddress(*addr, &ai, *privKey, *pow, "")

	return privKey
}
