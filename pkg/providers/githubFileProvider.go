package providers

import (
	"fmt"
	"github.com/zDev23/extension-patcher/pkg/utils"
	"os"
	"path/filepath"
	"strings"
)

func NewGithubFileProvider(uri string) *GithubFileProvider {
	return &GithubFileProvider{BaseProvider: BaseProvider{uri: uri}}
}

type GithubFileProvider struct {
	BaseProvider
}

func (s *GithubFileProvider) GetContent(sha256, dst string, keepOriginal bool) {
	parts := strings.Split(s.uri, "/")
	fileName := parts[len(parts)-1]
	dstFileName := filepath.Join(dst, fileName+".orig")
	if !utils.FileExists(dstFileName) {
		utils.CheckErr(downloadExtension(s, dstFileName))
		fmt.Println("extension downloaded")
	}
	utils.CheckErr(s.ValidateSha256(dstFileName, sha256))
	utils.CheckErr(utils.CopyFile(dstFileName, strings.TrimSuffix(dstFileName, ".orig")))
	if !keepOriginal {
		_ = os.Remove(dstFileName)
	}
}
