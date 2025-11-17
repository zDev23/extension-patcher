package providers

import (
	"errors"
	"github.com/ogame-ninja/extension-patcher/pkg/utils"
	"io"
	"strings"
)

const (
	chromeWebstorePrefix1 = "https://chrome.google.com/webstore/detail/"
	chromeWebstorePrefix2 = "https://chromewebstore.google.com/detail/"
	mozillaWebstorePrefix = "https://addons.mozilla.org/"
	openUserJSPrefix      = "https://openuserjs.org/install/"
	githubPrefix          = "https://github.com/"
	greasyForkPrefix      = "https://greasyfork.org/"
)

type Provider interface {
	GetName() string
	GetDownloadLink() string
	ValidatePayload(io.Reader) error
	GetContent(sha256, extensionName string, keepOriginal bool)
	ValidateSha256(path, expectedSha256 string) error
}

var ErrInvalidUri = errors.New("invalid uri")

func New(uri string) (Provider, error) {
	if strings.HasPrefix(uri, chromeWebstorePrefix1) ||
		strings.HasPrefix(uri, chromeWebstorePrefix2) {
		return NewChromeWebstore(uri), nil
	} else if strings.HasPrefix(uri, mozillaWebstorePrefix) {
		return NewMozillaWebstore(uri), nil
	} else if strings.HasPrefix(uri, openUserJSPrefix) {
		return NewOpenUserJS(uri), nil
	} else if strings.HasPrefix(uri, githubPrefix) {
		return NewGithubProvider(uri), nil
	} else if !strings.HasPrefix(uri, "http") {
		return NewFile(uri), nil
	}
	return nil, ErrInvalidUri
}

func MustNew(uri string) Provider {
	return utils.Must(New(uri))
}
