/*
Copyright 2021 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package sign

import (
	"bytes"
	"context"
	"crypto"
	"fmt"
	"time"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"golang.org/x/crypto/openpgp/packet"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/cloudkms/v1"
	"google.golang.org/api/option"

	"github.com/cert-manager/release/pkg/sign/internal/kmssigner"
)

const (
	pgpName  = "cert-manager Maintainers"
	pgpEmail = "cert-manager-maintainers@googlegroups.com"

	// PGP key ids are complicated, see https://datatracker.ietf.org/doc/html/rfc4880#section-12
	// Key IDs include hashed data taken from the "public key packet" (go doc "golang.org/x/crypto/openpgp/packet" PublicKey)
	// The public key packet, crucially, includes its CreationTime. That means that for a key with a stable
	// static ID to be created, the creation time must be static.
	// We could use the time that the KMS key was created, but that requires addtional permissions (i.e., the permission to "get"
	// the key using the GCP API), so instead we hardcode the creation time for all keys.
	// The time below is "2021-09-29T14:09:53Z", which just happens to be the time this change was started.
	keyCreationTimeUnix = 1632924593
)

// see comment for keyCreationTimeUnix
var staticKeyCreationTime = time.Unix(keyCreationTimeUnix, 0)

// PGPArmoredBlock is an ASCII-armored PGP key block
type PGPArmoredBlock string

// BootstrapPGPFromGCP creates a new PGP public key with a hardcoded cert-manager identity,
// signed using a named GCP KMS key. The KMS key can then be used for code signing, and the
// public key distributed for verification purposes.
func BootstrapPGPFromGCP(ctx context.Context, key string) (PGPArmoredBlock, error) {
	// Largely taken from:
	// https://github.com/heptiolabs/google-kms-pgp/blob/89c17dd5877a5c0f98f1906444b831dd2352b365/main.go#L131-L209
	entity, packetCfg, err := deriveEntity(ctx, key)
	if err != nil {
		return "", fmt.Errorf("failed to get an entity from key %q: %w", key, err)
	}

	// hardcode name + email, at least for now
	uid := packet.NewUserId(pgpName, "" /* no comment */, pgpEmail)
	if uid == nil {
		return "", fmt.Errorf("could not generate PGP user ID metadata; this indicates there were invalid characters in name or email")
	}

	isPrimary := true
	entity.Identities[uid.Id] = &openpgp.Identity{
		Name:   uid.Id,
		UserId: uid,
		SelfSignature: &packet.Signature{
			// SigTypePositiveCert means "we're absolutely sure this identity is correct"
			// Since we're the ones creating the identity, we're sure it's correct
			SigType: packet.SigTypePositiveCert,
			// CreationTime is informational; since we don't set a lifetime or expiry,
			// the key never expires.
			CreationTime: entity.PrimaryKey.CreationTime,
			PubKeyAlgo:   entity.PrimaryKey.PubKeyAlgo,
			Hash:         packetCfg.Hash(),
			IsPrimaryId:  &isPrimary,
			// FlagsValid must be set if any flags are set
			// FlagSign is true because the key is intended as a signing key
			// No other flags are relevant
			FlagsValid:  true,
			FlagSign:    true,
			IssuerKeyId: &entity.PrimaryKey.KeyId,
		},
	}

	if err := entity.Identities[uid.Id].SelfSignature.SignUserId(uid.Id, entity.PrimaryKey, entity.PrivateKey, packetCfg); err != nil {
		return "", fmt.Errorf("could not self-sign PGP public key: %w", err)
	}

	out := &bytes.Buffer{}

	armoredWriter, err := armor.Encode(out, "PGP PUBLIC KEY BLOCK", nil)
	if err != nil {
		return "", fmt.Errorf("could not create writer for public key: %w", err)
	}

	if err := entity.Serialize(armoredWriter); err != nil {
		return "", fmt.Errorf("could not serialize public key: %w", err)
	}

	if err := armoredWriter.Close(); err != nil {
		return "", fmt.Errorf("error closing public key writer: %w", err)
	}

	// Always add a blank line to the end of the raw output
	fmt.Fprintf(out, "\n")

	return PGPArmoredBlock(out.String()), nil
}

// deriveEntity creates a PGP entity from the given key; the entity wraps a private key and an
// identity and can be used for signing either the key itself or other charts. The openpgp packet
// config is also returned.
func deriveEntity(ctx context.Context, key string) (*openpgp.Entity, *packet.Config, error) {
	// Largely taken from:
	// https://github.com/heptiolabs/google-kms-pgp/blob/89c17dd5877a5c0f98f1906444b831dd2352b365/main.go#L320-L356

	// The DefaultHash needs to be SHA512 for helm
	cfg := &packet.Config{
		DefaultHash: crypto.SHA512,
	}

	if key == "" {
		return nil, nil, fmt.Errorf("missing required KMS key for deriving a PGP identity")
	}

	oauthClient, err := google.DefaultClient(ctx, cloudkms.CloudPlatformScope)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create GCP OAuth2 client: %w", err)
	}

	svc, err := cloudkms.NewService(ctx, option.WithHTTPClient(oauthClient))
	if err != nil {
		return nil, nil, fmt.Errorf("could not create GCP KMS client: %w", err)
	}

	signer, err := kmssigner.NewWithExplicitMetadata(svc, key, cfg.DefaultHash, staticKeyCreationTime)
	if err != nil {
		return nil, nil, fmt.Errorf("could not create KMS signer: %w", err)
	}

	entity := &openpgp.Entity{
		PrimaryKey: packet.NewRSAPublicKey(signer.CreationTime(), signer.RSAPublicKey()),
		PrivateKey: packet.NewSignerPrivateKey(signer.CreationTime(), signer),
		Identities: make(map[string]*openpgp.Identity),
	}

	entity.PrivateKey.PubKeyAlgo = packet.PubKeyAlgoRSA

	// The original states that this is required possibly because of a bug:
	// "Without this, signatures end up with a key ID that doesn't match the primary key"
	entity.PrivateKey.KeyId = entity.PrimaryKey.KeyId

	return entity, cfg, nil
}
