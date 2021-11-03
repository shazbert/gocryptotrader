package chain

import (
	"errors"
	"strings"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/currency"
)

// Item defines the chain type
type Item string

// Items defines a list of chain types
type Items []Item

// Translator defines an exchange translation mechanism from internal standard
// to exchange acceptable string
type Translator struct {
	m   map[Item]string
	mtx sync.RWMutex
}

const (
	ERC20 Item = "ERC20"
	TRT20 Item = "TRT20"
)

var (
	// ErrTranslationNotFound exported error for when there is no discernable
	// translation
	ErrTranslationNotFound = errors.New("translation not found")

	errTranslatorIsNil    = errors.New("translator is nil")
	errChainIsEmpty       = errors.New("incoming chain is empty")
	errTranslationIsEmpty = errors.New("incoming translation is empty")

	supported = Items{
		ERC20,
		TRT20,
	}
)

// Contains checks if it has itself bruh
func (i Items) Contains(c currency.Code) bool {
	for x := range i {
		if strings.EqualFold(string(i[x]), c.String()) {
			return true
		}
	}
	return false
}

// NewTranslator creates a new translator
func NewTranslator() *Translator {
	return &Translator{m: make(map[Item]string)}
}

func Load(newChain string) Item { return Item(newChain) }

// Load loads a translation
func (t *Translator) Load(app Item, translation string) error {
	if t == nil {
		return errTranslatorIsNil
	}
	if app == "" {
		return errChainIsEmpty
	}
	if translation == "" {
		return errTranslationIsEmpty
	}

	t.mtx.Lock()
	t.m[app] = translation
	t.mtx.Unlock()
	return nil
}

// Translate returns the string representation of the loaded chain item
func (t *Translator) Translate(app Item) (string, error) {
	if t == nil {
		return "", errTranslatorIsNil
	}
	if app == "" {
		return "", errChainIsEmpty
	}

	t.mtx.RLock()
	defer t.mtx.RUnlock()
	translation, ok := t.m[app]
	if !ok {
		return "", ErrTranslationNotFound
	}
	return translation, nil
}
