package extension_patcher

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"github.com/zDev23/extension-patcher/pkg/providers"
	"github.com/zDev23/extension-patcher/pkg/utils"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

const (
	perm              os.FileMode = 0644
	panicOnReplaceErr             = true
	jsExtension                   = ".js"
)

// Int re-export the Int function
var Int = utils.Int

type Processor func([]byte) []byte

type FileAndProcessors struct {
	FileName   string
	Processors []Processor
}

type Params struct {
	ExtensionName    string // infinity
	ExpectedSha256   string // 315738d9184062db0e42deddf6ab64268b4f7c522484892cf0abddf0560f6bcd
	Uri              string // https://chrome.google.com/webstore/detail/ogame-infinity/hfojakphgokgpbnejoobfamojbgolcbo
	Provider         providers.Provider
	Files            []FileAndProcessors
	JsBeautify       bool // Either or not to run "js-beautify" on js files
	DelayBeforeClose *int
	KeepOriginal     bool // Either or not to keep the original file
	AutoAnalysis     bool
}

type Patcher struct {
	params Params
}

func New(params Params) (*Patcher, error) {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	if params.Provider == nil && params.Uri != "" {
		params.Provider = providers.MustNew(params.Uri)
	}

	if params.ExpectedSha256 == "" {
		return nil, errors.New("missing ExpectedSha256")
	}
	if len(params.ExpectedSha256) != 0 && len(params.ExpectedSha256) != 64 {
		return nil, errors.New("ExpectedSha256 must be 64 characters long")
	}
	if params.Provider == nil {
		return nil, errors.New("missing Provider")
	}
	// No extension name provided, extract it from the webstore url
	params.ExtensionName = utils.Or(utils.Or(params.ExtensionName, params.Provider.GetName()), "patched")
	params.DelayBeforeClose = utils.Or(params.DelayBeforeClose, Int(5))
	if len(params.Files) == 0 {
		return nil, errors.New("missing Files")
	}
	return &Patcher{params: params}, nil
}

func MustNew(params Params) *Patcher {
	return utils.Must(New(params))
}

func (p *Patcher) Start() {
	delayBeforeClose := p.params.DelayBeforeClose
	if delayBeforeClose != nil {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println(r, "\n", string(debug.Stack()))
			} else {
				// This is useful on windows because the default CMD window closes immediately
				time.Sleep(time.Duration(*delayBeforeClose) * time.Second)
			}
		}()
	}

	extensionName := p.params.ExtensionName
	provider := p.params.Provider
	expectedSha256 := p.params.ExpectedSha256

	_ = os.Mkdir(extensionName, 0755)
	provider.GetContent(expectedSha256, extensionName, p.params.KeepOriginal)
	p.autoAnalyse()
	p.processFiles()

	path := utils.Must(os.Getwd())
	fmt.Println("Done. code generated in " + path)
}

func (p *Patcher) processFile(filename string, processors []Processor, maxLen int) {
	filePath := filepath.Join(p.params.ExtensionName, filename)
	by := utils.Must(os.ReadFile(filePath))
	for _, processor := range processors {
		by = processor(by)
	}

	if p.params.JsBeautify && strings.HasSuffix(filename, jsExtension) {
		by = JsBeautify(by)
	}

	utils.CheckErr(os.WriteFile(filePath, by, perm))
	fmt.Printf("%-"+strconv.Itoa(maxLen)+"v patched\n", filename)
}

func (p *Patcher) processFiles() {
	maxLen := 0
	for _, f := range p.params.Files {
		maxLen = max(len(f.FileName)+1, maxLen)
	}
	for _, f := range p.params.Files {
		p.processFile(f.FileName, f.Processors, maxLen)
	}
}

type myEntryStruct struct {
	os.DirEntry
	Prefix string
}

func (p *Patcher) autoAnalyse() {
	if !p.params.AutoAnalysis {
		return
	}
	fmt.Println(strings.Repeat("-", 80))
	extName := p.params.ExtensionName
	entries := utils.Must(os.ReadDir(extName))
	var myEntries []myEntryStruct
	for _, entry := range entries {
		myEntries = append(myEntries, myEntryStruct{DirEntry: entry, Prefix: extName})
	}
	var entry myEntryStruct
	for {
		if len(myEntries) == 0 {
			break
		}
		entry, myEntries = myEntries[0], myEntries[1:]
		filePath := filepath.Join(entry.Prefix, entry.Name())
		if entry.IsDir() {
			newEntries, err := os.ReadDir(filePath)
			if err != nil {
				log.Println(err)
				continue
			}
			for _, newEntry := range newEntries {
				myEntries = append(myEntries, myEntryStruct{DirEntry: newEntry, Prefix: filePath})
			}
			continue
		}
		by, err := os.ReadFile(filePath)
		if err != nil {
			log.Println(err)
			continue
		}
		analyzeFileContent(by, filePath, entry)
	}
	fmt.Println(strings.Repeat("-", 80))
}

func analyzeFileContent(by []byte, filePath string, entry myEntryStruct) {
	terms := []string{
		"index.php",
		"ogame.gameforge.com",
		"alliances.xml",
		"players.xml",
		"universe.xml",
		"highscore.xml",
		"location.host",
	}
	var termsFound []string
	for _, term := range terms {
		if bytes.Contains(by, []byte(term)) {
			termsFound = append(termsFound, Yellow(term))
		}
	}
	if len(termsFound) > 0 {
		processFileContent(by, filePath, entry, termsFound, terms)
	}
}

func processFileContent(by []byte, filePath string, entry myEntryStruct, termsFound []string, terms []string) {
	fmt.Printf("%s\n", Green(filePath))
	fmt.Printf("contains: %s\n", strings.Join(termsFound, ", "))

	// If we are processing a .js file, run js-beautify on it and save it
	if strings.HasSuffix(filePath, jsExtension) {
		by = JsBeautify(by)
		info, _ := entry.DirEntry.Info()
		if err := os.WriteFile(filePath, by, info.Mode()); err != nil {
			log.Println(err)
		}
	}

	scanner := bufio.NewScanner(bytes.NewReader(by))
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, len(by))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		if shouldProcessLine(line, terms) {
			line = processLine(line, terms)
			fmt.Printf("%d: %s\n", lineNumber, line)
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			log.Printf("%d: %v\n", lineNumber+1, err)
		} else {
			log.Println(err)
		}
	}
	fmt.Println(strings.Repeat("-", 30))
}

// We should process a line if any of the terms is present in it
func shouldProcessLine(line string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(line, term) {
			return true
		}
	}
	return false
}

// Colorify all the terms in the line
func processLine(line string, terms []string) string {
	var newTerms []string
	for _, term := range terms {
		newTerms = append(newTerms, term, Red(term))
	}
	replacer := strings.NewReplacer(newTerms...)
	return replacer.Replace(line)
}

// NoColor ...
var NoColor = utils.False()

// Terminal styling constants
const (
	knrm = "\x1B[0m"
	kred = "\x1B[31m"
	kgrn = "\x1B[32m"
	kyel = "\x1B[33m"
	kblu = "\x1B[34m"
	kmag = "\x1B[35m"
	kcyn = "\x1B[36m"
	kwht = "\x1B[37m"
)

func colorStr(color string, val string) string {
	if NoColor {
		return val
	}
	return color + val + knrm
}

func White(val string) string   { return colorStr(kwht, val) }
func Cyan(val string) string    { return colorStr(kcyn, val) }
func Red(val string) string     { return colorStr(kred, val) }
func Blue(val string) string    { return colorStr(kblu, val) }
func Yellow(val string) string  { return colorStr(kyel, val) }
func Green(val string) string   { return colorStr(kgrn, val) }
func Magenta(val string) string { return colorStr(kmag, val) }

func NewFile(fileName string, processors ...Processor) FileAndProcessors {
	return FileAndProcessors{FileName: fileName, Processors: processors}
}

func JsBeautify(in []byte) []byte {
	// -q, --quiet      Suppress logging to stdout
	// -f, --file       Input file(s) (Pass '-' for stdin)
	cmd := exec.Command("js-beautify", "-q", "-f '-'")
	cmd.Stdin = bytes.NewReader(in)
	return utils.Must(cmd.Output())
}

// MustReplace replace "n" occurrences of old with new
// If there are fewer occurrences of old than "n", panic
func MustReplace(in []byte, old, new string, n int) (out []byte) {
	return utils.First(mustReplace(in, old, new, n))
}

// MustReplaceN replace all "n" occurrences of old with new
// If there are fewer/more occurrences of old than n, panic
func MustReplaceN(in []byte, old, new string, n int) []byte {
	return mustReplaceN(in, old, new, n)
}

// Replace "n" occurrences of old with new
// Return the last index of the replaced text
func replace(in []byte, old, new string, n int) (out []byte, lastIdx int, err error) {
	oldBytes := []byte(old)
	newBytes := []byte(new)
	newBytes = bytes.Replace(newBytes, []byte("{old}"), oldBytes, 1)
	if n == -1 {
		out = bytes.Replace(in, oldBytes, newBytes, -1)
		return out, len(out), nil
	}
	replacementLength := len(new)
	for i := 0; i < n; i++ {
		startIdx := bytes.Index(in, oldBytes)
		if startIdx == -1 {
			return out, lastIdx, fmt.Errorf("expected %d replacements, did %d", n, i)
		}
		newOut := bytes.Replace(in, oldBytes, newBytes, 1)
		idx := startIdx + replacementLength
		out = append(out, newOut[:idx]...)
		in = newOut[idx:]
		lastIdx = len(out)
	}
	out = append(out, in...)
	return out, lastIdx, nil
}

func replaceN(in []byte, old, new string, n int) ([]byte, error) {
	out, lastIdx, err := replace(in, old, new, n)
	if err != nil {
		return out, err
	}
	count := bytes.Count(out[lastIdx:], []byte(old))
	if count > 0 {
		return out, errors.New("more text to replace " + strconv.Itoa(count))
	}
	return out, nil
}

func mustReplace(in []byte, old, new string, n int) (out []byte, lastIdx int) {
	out, lastIdx, err := replace(in, old, new, n)
	if err != nil {
		panic(err)
	}
	return out, lastIdx
}

func mustReplaceN(in []byte, old, new string, n int) []byte {
	out, err := replaceN(in, old, new, n)
	if err != nil {
		printOrPanic(err)
	}
	return out
}

func printOrPanic(err error) {
	if panicOnReplaceErr {
		panic(err)
	} else {
		fmt.Println(err.Error())
	}
}

// Helper functions, useful for tests
func mustReplaceStr(in, old, new string, n int) (out string) {
	return string(MustReplace([]byte(in), old, new, n))
}

// Helper functions, useful for tests
func mustReplaceNStr(in, old, new string, n int) string {
	return string(MustReplaceN([]byte(in), old, new, n))
}

// Helper functions, useful for tests
func replaceNStr(in, old, new string, n int) (string, error) {
	out, err := replaceN([]byte(in), old, new, n)
	return string(out), err
}
