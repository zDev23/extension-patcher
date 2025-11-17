package providers

import (
	"fmt"
	"github.com/zDev23/extension-patcher/pkg/utils"
	"os"
	"path/filepath"
	"strings"
)

func NewGreasyFork(uri string) *GreasyForkProvider {
	return &GreasyForkProvider{BaseProvider: BaseProvider{uri: uri}}
}

type GreasyForkProvider struct {
	BaseProvider
}

func (s *GreasyForkProvider) GetName() string {
	return getName(s.uri, `/install/[^/]+/([^.]+).user.js`)
}

func (s *GreasyForkProvider) GetContent(sha256, dst string, keepOriginal bool) {
	dstFileName := filepath.Join(dst, dst+".user.js.orig")
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
