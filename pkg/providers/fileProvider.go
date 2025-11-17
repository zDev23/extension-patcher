package providers

import (
	"github.com/zDev23/extension-patcher/pkg/utils"
	"path/filepath"
	"strings"
)

func NewFile(uri string) *FileProvider {
	return &FileProvider{BaseProvider: BaseProvider{uri: uri}}
}

type FileProvider struct {
	BaseProvider
}

func (s *FileProvider) GetContent(sha256, dst string, keepOriginal bool) {
	dstFileName := s.GetDownloadLink()
	if !utils.FileExists(dstFileName) {
		panic("file not found")
	}
	utils.CheckErr(s.ValidateSha256(dstFileName, sha256))
	fileName := filepath.Base(dstFileName)
	if !strings.HasSuffix(dstFileName, ".zip") {
		utils.CheckErr(utils.CopyFile(dstFileName, filepath.Join(dst, fileName)))
	} else {
		utils.CheckErr(utils.Unzip(dstFileName, dst))
	}
}
