package providers

import (
	"fmt"
	"github.com/zDev23/extension-patcher/pkg/utils"
	"os"
)

func NewMozillaWebstore(uri string) *MozillaWebstoreProvider {
	return &MozillaWebstoreProvider{BaseProvider: BaseProvider{uri: uri}}
}

type MozillaWebstoreProvider struct {
	BaseProvider
}

func (s *MozillaWebstoreProvider) GetName() string {
	return getName(s.uri, `/addon/([^/]+)/?`)
}

func (s *MozillaWebstoreProvider) GetDownloadLink() string {
	extensionID := getExtensionIDFromLink(s.uri)
	return "https://addons.mozilla.org/firefox/downloads/latest/" + extensionID + "/platform:3/" + extensionID + ".xpi"
}

func (s *MozillaWebstoreProvider) GetContent(sha256, dst string, keepOriginal bool) {
	dstFileName := s.GetName() + ".zip"
	if !utils.FileExists(dstFileName) {
		utils.CheckErr(downloadExtension(s, dstFileName))
		fmt.Println("extension downloaded")
	}
	utils.CheckErr(s.ValidateSha256(dstFileName, sha256))
	utils.CheckErr(utils.Unzip(dstFileName, dst))
	if !keepOriginal {
		_ = os.Remove(dstFileName)
	}
}
