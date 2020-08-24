package resolve

import (
	"crypto/sha256"
	"encoding/hex"
	"github.com/bitmaelum/bitmaelum-suite/internal/encrypt"
	"github.com/bitmaelum/bitmaelum-suite/pkg"
	"github.com/bitmaelum/bitmaelum-suite/pkg/address"
	"github.com/sirupsen/logrus"
)

// Service represents a resolver service tied to a specific repository
type Service struct {
	repo Repository
}

// Info is a structure returned by the external resolver system
type Info struct {
	Hash      string `json:"hash"`       // Hash of the email address
	PublicKey string `json:"public_key"` // Public key of the user
	Server    string `json:"server"`     // Server where this email address resides
}

// KeyRetrievalService initialises a key retrieval service.
func KeyRetrievalService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}

// Resolve resolves an address.
func (s *Service) Resolve(addr address.HashAddress) (*Info, error) {
	logrus.Debugf("Resolving %s", addr.String())
	info, err := s.repo.Resolve(addr)
	if err != nil {
		logrus.Debugf("Error while resolving %s: %s", addr.String(), err)
	}

	return info, err
}

// UploadInfo uploads resolve information to a service.
func (s *Service) UploadInfo(info pkg.Info, resolveAddress string) error {

	// @TODO: We maybe should sign with a different algo? Otherwise we use the same one for all systems
	privKey, err := encrypt.PEMToPrivKey([]byte(info.PrivKey))
	if err != nil {
		return err
	}

	// Sign resolve address
	hash := sha256.Sum256([]byte(resolveAddress))
	signature, err := encrypt.Sign(privKey, hash[:])
	if err != nil {
		return err
	}

	h, err := address.NewHash(info.Address)
	if err != nil {
		return err
	}

	// And upload
	return s.repo.Upload(
		*h,
		string(info.PubKey),
		resolveAddress,
		hex.EncodeToString(signature),
	)
}