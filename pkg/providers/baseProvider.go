package providers

import (
	"fmt"
	"github.com/zDev23/extension-patcher/pkg/utils"
	"io"
)

type BaseProvider struct {
	uri string
}

func (b BaseProvider) GetName() string                 { return "" }
func (b BaseProvider) GetDownloadLink() string         { return b.uri }
func (b BaseProvider) ValidatePayload(io.Reader) error { return nil }
func (b BaseProvider) GetContent(sha256, dst string, keepOriginal bool) {
	panic("implement me")
}
func (b BaseProvider) ValidateSha256(path, expectedSha256 string) error {
	extensionZipSha256 := utils.Sha256f(path)
	if extensionZipSha256 != expectedSha256 {
		return fmt.Errorf("invalid sha256 for %s (sha256: %s) \n", path, extensionZipSha256)
	}
	return nil
}
