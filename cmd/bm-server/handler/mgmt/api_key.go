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

package mgmt

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/bitmaelum/bitmaelum-suite/cmd/bm-server/handler"
	"github.com/bitmaelum/bitmaelum-suite/internal/apikey"
	"github.com/bitmaelum/bitmaelum-suite/internal/container"
	"github.com/bitmaelum/bitmaelum-suite/internal/parse"
)

type inputAPIKeyType struct {
	Permissions []string `json:"permissions"`
	Valid       string   `json:"valid"`
}

// NewAPIKey is a handler that will create a new API key (non-admin keys only)
func NewAPIKey(w http.ResponseWriter, req *http.Request) {
	key := handler.GetAPIKey(req)
	if !key.HasPermission(apikey.PermAPIKeys) {
		handler.ErrorOut(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input inputAPIKeyType
	err := handler.DecodeBody(w, req.Body, &input)
	if err != nil {
		handler.ErrorOut(w, http.StatusBadRequest, "incorrect body")
		return
	}

	// Our custom parser allows (and defaults) to using days
	validDuration, err := parse.ValidDuration(input.Valid)
	if err != nil {
		handler.ErrorOut(w, http.StatusBadRequest, "incorrect valid duration")
		return
	}

	err = parse.Permissions(input.Permissions)
	if err != nil {
		handler.ErrorOut(w, http.StatusBadRequest, "incorrect permissions")
		return
	}

	newAPIKey := apikey.NewKey(input.Permissions, validDuration)

	// Store API key into persistent storage
	repo := container.GetAPIKeyRepo()
	err = repo.Store(newAPIKey)
	if err != nil {
		msg := fmt.Sprintf("error while storing key: %s", err)
		handler.ErrorOut(w, http.StatusInternalServerError, msg)
		return
	}

	// Output key
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(jsonOut{
		"api_key": newAPIKey.ID,
	})
}
