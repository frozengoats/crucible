package ssh

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh/terminal"
)

type PassphraseProvider interface {
	GetPassphrase() (string, error)
}

type TypedPassphraseProvider struct {
}

func (p *TypedPassphraseProvider) GetPassphrase() (string, error) {
	// this is now an indication that this key is locked with a passphrase
	fmt.Printf("enter your passphrase: ")
	bytePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
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
