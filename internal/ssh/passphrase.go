package ssh

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

type PassphraseProvider interface {
	GetPassphrase() (string, error)
}

type TypedPassphraseProvider struct {
}

func (p *TypedPassphraseProvider) GetPassphrase() (string, error) {
	// this is now an indication that this key is locked with a passphrase
	// first check the environment for a passphrase, this is mostly used for debugging
	phrase := os.Getenv("CRUCIBLE_SSH_KEY_PASSPHRASE")
	if len(phrase) > 0 {
		return phrase, nil
	}

	fmt.Printf("enter your ssh key passphrase: ")
	bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Printf("\n")
	if err != nil {
		return "", err
	}
	passPhrase := string(bytePassword)
	return strings.Trim(passPhrase, "\n"), nil
}

func NewTypedPassphraseProvider() *TypedPassphraseProvider {
	return &TypedPassphraseProvider{}
}

type DefaultPassphraseProvider struct {
	passphrase string
}

func (p *DefaultPassphraseProvider) GetPassphrase() (string, error) {
	return p.passphrase, nil
}

func NewDefaultPassphraseProvider(passphrase string) *DefaultPassphraseProvider {
	return &DefaultPassphraseProvider{
		passphrase: passphrase,
	}
}
