package providers

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/zDev23/extension-patcher/pkg/utils"
	"io"
	"os"
)

func NewChromeWebstore(uri string) *ChromeWebstoreProvider {
	return &ChromeWebstoreProvider{BaseProvider: BaseProvider{uri: uri}}
}

type ChromeWebstoreProvider struct {
	BaseProvider
}

func (s *ChromeWebstoreProvider) GetDownloadLink() string {
	extensionID := getExtensionIDFromLink(s.uri)
	return buildDownloadLink(extensionID)
}

func (s *ChromeWebstoreProvider) GetName() string {
	return getName(s.uri, `/detail/([^/]+)/`)
}

func (s *ChromeWebstoreProvider) ValidatePayload(reader io.Reader) error {
	const magicBytes = 0x43723234 // Cr24
	var magic uint32
	if err := binary.Read(reader, binary.BigEndian, &magic); err != nil {
		return err
	}
	if magic != magicBytes {
		return InvalidMagicBytesErr
	}
	var version uint32
	if err := binary.Read(reader, binary.BigEndian, &version); err != nil {
		return err
	}
	var headerLength uint32
	if err := binary.Read(reader, binary.LittleEndian, &headerLength); err != nil {
		return err
	}
	buf := make([]byte, headerLength)
	if _, err := reader.Read(buf); err != nil {
		return err
	}
	return nil
}

func buildDownloadLink(extensionID string) string {
	return "https://clients2.google.com/service/update2/crx?" +
		"response=redirect&prodversion=131.0.0.0&acceptformat=crx3&" +
		"x=id%3D" + extensionID + "%26installsource%3Dondemand%26uc"
}

var InvalidMagicBytesErr = errors.New("invalid magic bytes")

func (s *ChromeWebstoreProvider) GetContent(sha256, dst string, keepOriginal bool) {
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
