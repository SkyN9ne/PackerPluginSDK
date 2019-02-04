package common

import (
	"bytes"
	"crypto/rand"
	"errors"
	"strconv"
	"testing"

	"github.com/hashicorp/packer/common/uuid"
	"golang.org/x/crypto/ssh"
)

// expected contains the data that the key pair should contain.
type expected struct {
	kind sshKeyPairType
	bits int
	desc string
	name string
	data []byte
}

func (o expected) matches(kp sshKeyPair) error {
	if o.kind.String() == "" {
		return errors.New("expected kind's value cannot be empty")
	}

	if o.bits <= 0 {
		return errors.New("expected bits' value cannot be less than or equal to 0")
	}

	if o.desc == "" {
		return errors.New("expected description's value cannot be empty")
	}

	if len(o.data) == 0 {
		return errors.New("expected random data value cannot be nothing")
	}

	if kp.Type() != o.kind {
		return errors.New("key pair type should be " + o.kind.String() +
			" - got '" + kp.Type().String() + "'")
	}

	if kp.Bits() != o.bits {
		return errors.New("key pair bits should be " + strconv.Itoa(o.bits) +
			" - got " + strconv.Itoa(kp.Bits()))
	}

	if len(o.name) > 0 && kp.Name() != o.name {
		return errors.New("key pair name should be '" + o.name +
			"' - got '" + kp.Name() + "'")
	}

	expDescription := kp.Type().String() + " " + strconv.Itoa(o.bits)
	if kp.Description() != expDescription {
		return errors.New("key pair description should be '" +
			expDescription + "' - got '" + kp.Description() + "'")
	}

	err := o.verifyPublicKeyAuthorizedKeysFormat(kp)
	if err != nil {
		return err
	}

	err = o.verifySshKeyPair(kp)
	if err != nil {
		return err
	}

	return nil
}

func (o expected) verifyPublicKeyAuthorizedKeysFormat(kp sshKeyPair) error {
	newLines := []newLineOption{
		unixNewLine,
		noNewLine,
		windowsNewLine,
	}

	for _, nl := range newLines {
		publicKeyAk := kp.PublicKeyAuthorizedKeysFormat(nl)

		if len(publicKeyAk) < 2 {
			return errors.New("expected public key in authorized keys format to be at least 2 bytes")
		}

		switch nl {
		case noNewLine:
			if publicKeyAk[len(publicKeyAk) - 1] == '\n' {
				return errors.New("public key in authorized keys format has trailing new line when none was specified")
			}
		case unixNewLine:
			if publicKeyAk[len(publicKeyAk) - 1] != '\n' {
				return errors.New("public key in authorized keys format does not have unix new line when unix was specified")
			}
			if string(publicKeyAk[len(publicKeyAk) - 2:]) == windowsNewLine.String() {
				return errors.New("public key in authorized keys format has windows new line when unix was specified")
			}
		case windowsNewLine:
			if string(publicKeyAk[len(publicKeyAk) - 2:]) != windowsNewLine.String() {
				return errors.New("public key in authorized keys format does not have windows new line when windows was specified")
			}
		}

		if len(o.name) > 0 {
			if len(publicKeyAk) < len(o.name) {
				return errors.New("public key in authorized keys format is shorter than the key pair's name")
			}

			suffix := []byte{' '}
			suffix = append(suffix, o.name...)
			suffix = append(suffix, nl.Bytes()...)
			if !bytes.HasSuffix(publicKeyAk, suffix) {
				return errors.New("public key in authorized keys format with name does not have name in suffix - got '" +
					string(publicKeyAk) + "'")
			}
		}
	}

	return nil
}

func (o expected) verifySshKeyPair(kp sshKeyPair) error {
	signer, err := ssh.ParsePrivateKey(kp.PrivateKeyPemBlock())
	if err != nil {
		return errors.New("failed to parse private key during verification - " + err.Error())
	}

	signature, err := signer.Sign(rand.Reader, o.data)
	if err != nil {
		return errors.New("failed to sign test data during verification - " + err.Error())
	}

	err = signer.PublicKey().Verify(o.data, signature)
	if err != nil {
		return errors.New("failed to verify test data - " + err.Error())
	}

	return nil
}

func TestDefaultSshKeyPairBuilder_Build_Default(t *testing.T) {
	kp, err := newSshKeyPairBuilder().Build()
	if err != nil {
		t.Fatal(err.Error())
	}

	err = expected{
		kind: ecdsaSsh,
		bits: 521,
		desc: "ecdsa 521",
		data: []byte(uuid.TimeOrderedUUID()),
	}.matches(kp)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestDefaultSshKeyPairBuilder_Build_EcdsaDefault(t *testing.T) {
	kp, err := newSshKeyPairBuilder().
		SetType(ecdsaSsh).
		Build()
	if err != nil {
		t.Fatal(err.Error())
	}

	err = expected{
		kind: ecdsaSsh,
		bits: 521,
		desc: "ecdsa 521",
		data: []byte(uuid.TimeOrderedUUID()),
	}.matches(kp)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestDefaultSshKeyPairBuilder_Build_RsaDefault(t *testing.T) {
	kp, err := newSshKeyPairBuilder().
		SetType(rsaSsh).
		Build()
	if err != nil {
		t.Fatal(err.Error())
	}

	err = expected{
		kind: rsaSsh,
		bits: 4096,
		desc: "rsa 4096",
		data: []byte(uuid.TimeOrderedUUID()),
	}.matches(kp)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestDefaultSshKeyPairBuilder_Build_NamedEcdsa(t *testing.T) {
	name := uuid.TimeOrderedUUID()

	kp, err := newSshKeyPairBuilder().
		SetType(ecdsaSsh).
		SetName(name).
		Build()
	if err != nil {
		t.Fatal(err.Error())
	}

	err = expected{
		kind: ecdsaSsh,
		bits: 521,
		desc: "ecdsa 521",
		data: []byte(uuid.TimeOrderedUUID()),
		name: name,
	}.matches(kp)
	if err != nil {
		t.Fatal(err.Error())
	}
}

func TestDefaultSshKeyPairBuilder_Build_NamedRsa(t *testing.T) {
	name := uuid.TimeOrderedUUID()

	kp, err := newSshKeyPairBuilder().
		SetType(rsaSsh).
		SetName(name).
		Build()
	if err != nil {
		t.Fatal(err.Error())
	}

	err = expected{
		kind: rsaSsh,
		bits: 4096,
		desc: "rsa 4096",
		data: []byte(uuid.TimeOrderedUUID()),
		name: name,
	}.matches(kp)
	if err != nil {
		t.Fatal(err.Error())
	}
}
