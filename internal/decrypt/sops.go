/*
SPDX-FileCopyrightText: 2026 The Flux authors
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

/*
Disclaimer: this file borrows code from
https://github.com/fluxcd/kustomize-controller/blob/main/internal/decryptor/decryptor.go
https://github.com/fluxcd/kustomize-controller/tree/main/internal/sops/keyservice
*/

package decrypt

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"

	"filippo.io/age"

	sops "github.com/getsops/sops/v3"
	sopsaes "github.com/getsops/sops/v3/aes"
	sopsage "github.com/getsops/sops/v3/age"
	sopscommon "github.com/getsops/sops/v3/cmd/sops/common"
	sopsformats "github.com/getsops/sops/v3/cmd/sops/formats"
	sopsconfig "github.com/getsops/sops/v3/config"
	sopskeys "github.com/getsops/sops/v3/keys"
	sopskeyservice "github.com/getsops/sops/v3/keyservice"
	sopslogging "github.com/getsops/sops/v3/logging"
	sopspgp "github.com/getsops/sops/v3/pgp"
)

// TODO: needs refactoring and testing

const (
	unsupportedFormat = sopsformats.Format(-1)

	decryptionPGPExt = ".asc"
	decryptionAgeExt = ".agekey"
)

var (
	sopsFormatToString = map[sopsformats.Format]string{
		sopsformats.Dotenv: "dotenv",
		sopsformats.Ini:    "INI",
		sopsformats.Json:   "JSON",
		sopsformats.Yaml:   "YAML",
	}
	sopsFormatToMarkerBytes = map[sopsformats.Format][]byte{
		sopsformats.Dotenv: []byte("sops_mac=ENC["),
		sopsformats.Ini:    []byte("[sops]"),
		sopsformats.Json:   []byte("\"mac\": \"ENC["),
		sopsformats.Yaml:   []byte("mac: ENC["),
	}
)

type SopsDecryptor struct {
	gnuPGHome     sopspgp.GnuPGHome
	ageIdentities sopsage.ParsedIdentities
	keyServices   []sopskeyservice.KeyServiceClient
}

var _ Decryptor = &SopsDecryptor{}

func NewSopsDecryptor(keys map[string][]byte) (_ *SopsDecryptor, err error) {
	gnuPGHome, err := os.MkdirTemp("", "gpg-")
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			os.RemoveAll(gnuPGHome)
		}
	}()
	decryptor := &SopsDecryptor{gnuPGHome: sopspgp.GnuPGHome(gnuPGHome)}
	for name, value := range keys {
		switch filepath.Ext(name) {
		case decryptionPGPExt:
			if err = decryptor.gnuPGHome.Import(value); err != nil {
				return nil, err
			}
		case decryptionAgeExt:
			if err = decryptor.ageIdentities.Import(string(value)); err != nil {
				return nil, err
			}
		}
	}
	serverOpts := []ServerOption{
		WithGnuPGHome(decryptor.gnuPGHome),
		WithAgeIdentities(decryptor.ageIdentities),
	}
	server := NewServer(serverOpts...)
	decryptor.keyServices = append(decryptor.keyServices, sopskeyservice.NewCustomLocalClient(server))
	return decryptor, nil
}

func (d *SopsDecryptor) Decrypt(input []byte, path string) ([]byte, error) {
	inputFormat := detectFormatFromMarkerBytes(input)
	if inputFormat == unsupportedFormat {
		return input, nil
	}
	outputFormat := sopsformats.FormatForPath(path)

	return d.SopsDecryptWithFormat(input, inputFormat, outputFormat)
}

func (d *SopsDecryptor) SopsDecryptWithFormat(input []byte, inputFormat sopsformats.Format, outputFormat sopsformats.Format) (_ []byte, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to emit encrypted %s file as decrypted %s: %v",
				sopsFormatToString[inputFormat], sopsFormatToString[outputFormat], r)
		}
	}()

	store := sopscommon.StoreForFormat(inputFormat, sopsconfig.NewStoresConfig())

	tree, err := store.LoadEncryptedFile(input)
	if err != nil {
		return nil, sopsUserErr(fmt.Sprintf("failed to load encrypted %s data", sopsFormatToString[inputFormat]), err)
	}

	for _, group := range tree.Metadata.KeyGroups {
		sort.SliceStable(group, func(i, j int) bool {
			return isOfflineMethod(group[i]) && !isOfflineMethod(group[j])
		})
	}

	metadataKey, err := tree.Metadata.GetDataKeyWithKeyServices(d.keyServices, sops.DefaultDecryptionOrder)
	if err != nil {
		return nil, sopsUserErr("cannot get sops data key", err)
	}

	cipher := sopsaes.NewCipher()
	if _, err := tree.Decrypt(metadataKey, cipher); err != nil {
		return nil, sopsUserErr("error decrypting sops tree", err)
	}

	if seemsBinary(&tree) {
		outputFormat = sopsformats.Binary
	}

	outputStore := sopscommon.StoreForFormat(outputFormat, sopsconfig.NewStoresConfig())
	out, err := outputStore.EmitPlainFile(tree.Branches)
	if err != nil {
		return nil, sopsUserErr(fmt.Sprintf("failed to emit encrypted %s file as decrypted %s",
			sopsFormatToString[inputFormat], sopsFormatToString[outputFormat]), err)
	}
	return out, nil
}

func (d *SopsDecryptor) Cleanup() {
	if d.gnuPGHome != "" {
		os.RemoveAll(d.gnuPGHome.String())
	}
}

func detectFormatFromMarkerBytes(b []byte) sopsformats.Format {
	for k, v := range sopsFormatToMarkerBytes {
		if bytes.Contains(b, v) {
			return k
		}
	}
	return unsupportedFormat
}
func seemsBinary(tree *sops.Tree) bool {
	if len(tree.Branches[0]) != 1 {
		return false
	}
	if tree.Branches[0][0].Key != "data" {
		return false
	}
	if _, ok := tree.Branches[0][0].Value.(string); !ok {
		return false
	}
	return true
}

func sopsUserErr(msg string, err error) error {
	if userErr, ok := err.(sops.UserError); ok {
		err = errors.New(userErr.UserError())
	}
	return errors.Wrap(err, msg)
}

func isOfflineMethod(mk sopskeys.MasterKey) bool {
	switch mk.(type) {
	case *sopspgp.MasterKey, *sopsage.MasterKey:
		return true
	default:
		return false
	}
}

type Server struct {
	gnuPGHome     sopspgp.GnuPGHome
	ageIdentities sopsage.ParsedIdentities
	defaultServer sopskeyservice.KeyServiceServer
}

func NewServer(options ...ServerOption) sopskeyservice.KeyServiceServer {
	s := &Server{}
	for _, opt := range options {
		opt.ApplyToServer(s)
	}

	if s.defaultServer == nil {
		s.defaultServer = &sopskeyservice.Server{
			Prompt: false,
		}
	}

	sopslogging.SetLevel(0)

	return s
}

func (ks Server) Encrypt(ctx context.Context, req *sopskeyservice.EncryptRequest) (*sopskeyservice.EncryptResponse, error) {
	key := req.Key
	switch k := key.KeyType.(type) {
	case *sopskeyservice.Key_PgpKey:
		ciphertext, err := ks.encryptWithPgp(k.PgpKey, req.Plaintext)
		if err != nil {
			return nil, err
		}
		return &sopskeyservice.EncryptResponse{
			Ciphertext: ciphertext,
		}, nil
	case *sopskeyservice.Key_AgeKey:
		ciphertext, err := ks.encryptWithAge(k.AgeKey, req.Plaintext)
		if err != nil {
			return nil, err
		}
		return &sopskeyservice.EncryptResponse{
			Ciphertext: ciphertext,
		}, nil
	case nil:
		return nil, fmt.Errorf("must provide a key")
	}
	return ks.defaultServer.Encrypt(ctx, req)
}

func (ks Server) Decrypt(ctx context.Context, req *sopskeyservice.DecryptRequest) (*sopskeyservice.DecryptResponse, error) {
	key := req.Key
	switch k := key.KeyType.(type) {
	case *sopskeyservice.Key_PgpKey:
		plaintext, err := ks.decryptWithPgp(k.PgpKey, req.Ciphertext)
		if err != nil {
			return nil, err
		}
		return &sopskeyservice.DecryptResponse{
			Plaintext: plaintext,
		}, nil
	case *sopskeyservice.Key_AgeKey:
		plaintext, err := ks.decryptWithAge(k.AgeKey, req.Ciphertext)
		if err != nil {
			return nil, err
		}
		return &sopskeyservice.DecryptResponse{
			Plaintext: plaintext,
		}, nil
	case nil:
		return nil, fmt.Errorf("must provide a key")
	}
	return ks.defaultServer.Decrypt(ctx, req)
}

func (ks *Server) encryptWithPgp(key *sopskeyservice.PgpKey, plaintext []byte) ([]byte, error) {
	pgpKey := sopspgp.NewMasterKeyFromFingerprint(key.Fingerprint)
	sopspgp.DisableOpenPGP{}.ApplyToMasterKey(pgpKey)
	if ks.gnuPGHome != "" {
		ks.gnuPGHome.ApplyToMasterKey(pgpKey)
	}
	err := pgpKey.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}
	return []byte(pgpKey.EncryptedKey), nil
}

func (ks *Server) decryptWithPgp(key *sopskeyservice.PgpKey, ciphertext []byte) ([]byte, error) {
	pgpKey := sopspgp.NewMasterKeyFromFingerprint(key.Fingerprint)
	sopspgp.DisableOpenPGP{}.ApplyToMasterKey(pgpKey)
	if ks.gnuPGHome != "" {
		ks.gnuPGHome.ApplyToMasterKey(pgpKey)
	}
	pgpKey.EncryptedKey = string(ciphertext)
	plaintext, err := pgpKey.Decrypt()
	return plaintext, err
}

func (ks Server) encryptWithAge(key *sopskeyservice.AgeKey, plaintext []byte) ([]byte, error) {
	ageKey := sopsage.MasterKey{
		Recipient: key.Recipient,
	}
	if err := ageKey.Encrypt(plaintext); err != nil {
		return nil, err
	}
	return []byte(ageKey.EncryptedKey), nil
}

func (ks *Server) decryptWithAge(key *sopskeyservice.AgeKey, ciphertext []byte) ([]byte, error) {
	ageKey := sopsage.MasterKey{
		Recipient: key.Recipient,
	}
	ks.ageIdentities.ApplyToMasterKey(&ageKey)
	ageKey.EncryptedKey = string(ciphertext)
	plaintext, err := ageKey.Decrypt()
	return plaintext, err
}

type ServerOption interface {
	ApplyToServer(s *Server)
}

type WithGnuPGHome string

func (o WithGnuPGHome) ApplyToServer(s *Server) {
	s.gnuPGHome = sopspgp.GnuPGHome(o)
}

type WithAgeIdentities []age.Identity

func (o WithAgeIdentities) ApplyToServer(s *Server) {
	s.ageIdentities = sopsage.ParsedIdentities(o)
}
