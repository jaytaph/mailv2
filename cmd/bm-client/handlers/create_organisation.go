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

package handlers

import (
	"context"
	"fmt"
	"net"
	"os"

	"github.com/bitmaelum/bitmaelum-suite/cmd/bm-client/internal"
	"github.com/bitmaelum/bitmaelum-suite/cmd/bm-client/internal/container"
	"github.com/bitmaelum/bitmaelum-suite/cmd/bm-client/internal/stepper"
	bminternal "github.com/bitmaelum/bitmaelum-suite/internal"
	"github.com/bitmaelum/bitmaelum-suite/internal/organisation"
	"github.com/bitmaelum/bitmaelum-suite/internal/vault"
	"github.com/bitmaelum/bitmaelum-suite/pkg/bmcrypto"
	"github.com/bitmaelum/bitmaelum-suite/pkg/hash"
	pow "github.com/bitmaelum/bitmaelum-suite/pkg/proofofwork"
)

const (
	ctxOrganisationFound ctxKey = iota
	ctxOrgVault
	ctxOrgAddr
	ctxOrgHash
	ctxOrgValidations
	ctxOrgValidationsStr
	ctxOrgKeyType
	ctxOrgKeyPair
	ctxOrgName
	ctxOrgInfo
	ctxOrgProof
	ctxOrgDomains
	ctxOrgReserved
)

// CreateOrganisation creates a new organisation locally in the vault and pushes the public key to the resolver
func CreateOrganisation(v *vault.Vault, orgAddr, fullName string, orgValidations []string, kt bmcrypto.KeyType) {
	s := stepper.New()

	// Set some initial values in the context. We read and write to the context to deal with variables instead of using globals.
	s.Ctx = context.WithValue(s.Ctx, ctxOrgVault, v)
	s.Ctx = context.WithValue(s.Ctx, ctxOrgKeyType, kt)
	s.Ctx = context.WithValue(s.Ctx, ctxOrgName, fullName)
	s.Ctx = context.WithValue(s.Ctx, ctxOrgValidationsStr, orgValidations)
	s.Ctx = context.WithValue(s.Ctx, ctxOrgAddr, orgAddr)
	s.Ctx = context.WithValue(s.Ctx, ctxOrgHash, hash.New(orgAddr))

	// Add all the steps from the account creation procedure

	s.AddStep(stepper.Step{
		Title:   "Checking if organisation is already known in the resolver service",
		RunFunc: checkOrganisationInResolver,
	})

	s.AddStep(stepper.Step{
		Title:   "Checking if address is a reserved name",
		RunFunc: checkOrganisationReservedAddress,
	})

	s.AddStep(stepper.Step{
		Title:          "Checking if validations are correct",
		DisplaySpinner: true,
		RunFunc:        checkValidations,
	})

	s.AddStep(stepper.Step{
		Title:   "Checking if the organisation is already present in the vault",
		RunFunc: checkOrganisationInVault,
	})

	s.AddStep(stepper.Step{
		Title:          "Generating organisation public/private keypair",
		DisplaySpinner: true,
		SkipIfFunc:     organisationNotFoundInContext,
		RunFunc:        generateOrganisationKeyPair,
	})

	s.AddStep(stepper.Step{
		Title:          fmt.Sprintf("Doing some work to let people know this is not a fake account, %sthis might take a while%s...", stepper.AnsiFgYellow, stepper.AnsiReset),
		DisplaySpinner: true,
		SkipIfFunc:     organisationNotFoundInContext,
		RunFunc:        doProofOfWorkOrg,
	})

	s.AddStep(stepper.Step{
		Title:      "Placing your new organisation into the vault",
		SkipIfFunc: organisationNotFoundInContext,
		RunFunc:    addOrganisationToVault,
	})

	s.AddStep(stepper.Step{
		Title:          "Checking domains for reservation proof",
		RunFunc:        checkOrganisationReservedDomains,
		SkipIfFunc:     func(s stepper.Stepper) bool { return s.Ctx.Value(ctxOrgReserved).(bool) == false },
		DisplaySpinner: true,
	})

	s.AddStep(stepper.Step{
		Title:          "Making your organisation known to the outside world",
		DisplaySpinner: true,
		RunFunc:        uploadOrganisationToResolver,
	})

	// Run the stepper
	s.Run()
	if s.Status == stepper.FAILURE {
		fmt.Println("There was an error while creating the organisation.")
		os.Exit(1)
	}

	info := s.Ctx.Value(ctxOrgInfo).(*vault.OrganisationInfo)
	kp := info.GetActiveKey().KeyPair
	mnemonic := bminternal.WordWrap(bmcrypto.GetMnemonic(&kp), 78)

	fmt.Println(internal.GenerateFromMnemonicTemplate(internal.OrganisationCreatedTemplate, mnemonic))
}

func checkOrganisationInVault(s *stepper.Stepper) stepper.StepResult {
	v := s.Ctx.Value(ctxOrgVault).(*vault.Vault)
	orgHash := s.Ctx.Value(ctxOrgHash).(hash.Hash)

	if !v.HasOrganisation(orgHash) {
		return stepper.StepResult{
			Status:  stepper.SUCCESS,
			Message: "not found. That's good.",
		}
	}

	info, err := v.GetOrganisationInfo(orgHash)
	if err != nil {
		return stepper.StepResult{
			Status:  stepper.FAILURE,
			Message: "found. But error while fetching from the vault.",
		}
	}

	s.Ctx = context.WithValue(s.Ctx, ctxOrgInfo, info)
	s.Ctx = context.WithValue(s.Ctx, ctxOrganisationFound, true)

	return stepper.StepResult{
		Status:  stepper.SUCCESS,
		Message: "found. That's odd, but let's continue...",
	}
}

func checkOrganisationInResolver(s *stepper.Stepper) stepper.StepResult {
	orgHash := s.Ctx.Value(ctxOrgHash).(hash.Hash)

	ks := container.Instance.GetResolveService()
	_, err := ks.ResolveOrganisation(orgHash)

	if err == nil {
		return stepper.StepResult{
			Status:  stepper.FAILURE,
			Message: "organisation already found",
		}
	}

	return stepper.StepResult{
		Status: stepper.SUCCESS,
	}
}

func checkValidations(s *stepper.Stepper) stepper.StepResult {
	arr := s.Ctx.Value(ctxOrgValidationsStr).([]string)
	validations, err := organisation.NewValidationTypeFromStringArray(arr)
	if err != nil {
		return stepper.StepResult{
			Status:  stepper.FAILURE,
			Message: "validation failed",
		}
	}

	s.Ctx = context.WithValue(s.Ctx, ctxOrgValidations, validations)

	return stepper.StepResult{
		Status: stepper.SUCCESS,
	}
}

func doProofOfWorkOrg(s *stepper.Stepper) stepper.StepResult {
	orgHash := s.Ctx.Value(ctxOrgHash).(hash.Hash)

	// Find the number of bits for address creation
	res := container.Instance.GetResolveService()
	resolverCfg := res.GetConfig()

	proof := pow.NewWithoutProof(resolverCfg.ProofOfWork.Organisation, orgHash.String())
	proof.WorkMulticore()

	s.Ctx = context.WithValue(s.Ctx, ctxOrgProof, proof)

	return stepper.StepResult{
		Status: stepper.SUCCESS,
	}
}

func generateOrganisationKeyPair(s *stepper.Stepper) stepper.StepResult {
	kt := s.Ctx.Value(ctxOrgKeyType).(bmcrypto.KeyType)
	kp, err := bmcrypto.GenerateKeypairWithRandomSeed(kt)
	if err != nil {
		return stepper.StepResult{
			Status:  stepper.FAILURE,
			Message: err.Error(),
		}
	}

	s.Ctx = context.WithValue(s.Ctx, ctxOrgKeyPair, kp)
	return stepper.StepResult{
		Status: stepper.SUCCESS,
	}
}

func addOrganisationToVault(s *stepper.Stepper) stepper.StepResult {
	v := s.Ctx.Value(ctxOrgVault).(*vault.Vault)
	orgAddr := s.Ctx.Value(ctxOrgAddr).(string)
	name := s.Ctx.Value(ctxOrgName).(string)
	kp := s.Ctx.Value(ctxOrgKeyPair).(*bmcrypto.KeyPair)
	proof := s.Ctx.Value(ctxOrgProof).(*pow.ProofOfWork)
	validations := s.Ctx.Value(ctxOrgValidations).([]organisation.ValidationType)

	info := &vault.OrganisationInfo{
		Addr:     orgAddr,
		FullName: name,
		Keys: []vault.KeyPair{
			{
				KeyPair: *kp,
				Active:  true,
			},
		},
		Pow:         proof,
		Validations: validations,
	}

	v.AddOrganisation(*info)
	err := v.Persist()
	if err != nil {
		return stepper.StepResult{
			Status:  stepper.FAILURE,
			Message: fmt.Sprintf("error while saving organisation into vault: %#v", err),
		}
	}

	s.Ctx = context.WithValue(s.Ctx, ctxOrgInfo, info)
	s.Ctx = context.WithValue(s.Ctx, ctxOrganisationFound, true)

	return stepper.StepResult{
		Status: stepper.SUCCESS,
	}
}

func uploadOrganisationToResolver(s *stepper.Stepper) stepper.StepResult {
	info := s.Ctx.Value(ctxOrgInfo).(*vault.OrganisationInfo)

	ks := container.Instance.GetResolveService()
	err := ks.UploadOrganisationInfo(*info)
	if err != nil {
		return stepper.StepResult{
			Status:  stepper.FAILURE,
			Message: fmt.Sprintf("error while uploading organisation to the resolver: %s", err.Error()),
		}
	}

	return stepper.StepResult{
		Status: stepper.SUCCESS,
	}
}

func organisationNotFoundInContext(s stepper.Stepper) bool {
	return s.Ctx.Value(ctxOrganisationFound) == nil
}

func checkOrganisationReservedAddress(s *stepper.Stepper) stepper.StepResult {
	orgAddr := s.Ctx.Value(ctxOrgAddr).(string)
	orgHash := hash.New(orgAddr)

	ks := container.Instance.GetResolveService()
	domains, _ := ks.CheckReserved(orgHash)

	s.Ctx = context.WithValue(s.Ctx, ctxOrgReserved, len(domains) > 0)
	s.Ctx = context.WithValue(s.Ctx, ctxOrgDomains, domains)

	if len(domains) > 0 {
		return stepper.StepResult{
			Status:  stepper.NOTICE,
			Message: "Yes. DNS verification is needed in order to register this organisation",
		}
	}

	return stepper.StepResult{
		Status:  stepper.SUCCESS,
		Message: "Not reserved",
	}
}

func checkOrganisationReservedDomains(s *stepper.Stepper) stepper.StepResult {
	var kp *bmcrypto.KeyPair

	af := s.Ctx.Value(ctxOrganisationFound) != nil
	if af {
		info := s.Ctx.Value(ctxOrgInfo).(*vault.OrganisationInfo)
		k := info.GetActiveKey().KeyPair
		kp = &k
	} else {
		kp = s.Ctx.Value(ctxOrgKeyPair).(*bmcrypto.KeyPair)
	}

	domains := s.Ctx.Value(ctxOrgDomains).([]string)

	for _, domain := range domains {
		// Check domain
		entries, err := net.LookupTXT("_bitmaelum." + domain)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry == kp.PubKey.Fingerprint() {
				return stepper.StepResult{
					Status:  stepper.SUCCESS,
					Message: "found reservation at " + domain,
				}
			}
		}
	}

	messageTemplate := `could not find proof in the DNS.

In order to register this reserved organisation, make sure you add the following information to the DNS:

    _bitmaelum TXT {{ .Fingerprint }}

This entry could be added to any of the following domains: {{ .Domains }}. Once we have found the entry, we can 
register the organisation onto the keyserver. For more information, please visit https://bitmaelum.com/reserved
`

	msg := internal.GenerateFromFingerprintTemplate(messageTemplate, kp.PubKey.Fingerprint(), domains)

	return stepper.StepResult{
		Status:  stepper.FAILURE,
		Message: msg,
	}
}
