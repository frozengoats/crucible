package ssh

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
)

type PassphraseProvider interface {
	GetPassphrase() (string, error)
}

type TypedPassphraseProvider struct {
}

func (p *TypedPassphraseProvider) GetPassphrase() (string, error) {
	// this is now an indication that this key is locked with a passphrase
	fmt.Printf("enter your passphrase: ")
	reader := bufio.NewReader(os.Stdin)
	passPhrase, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}

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
