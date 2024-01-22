package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"strconv"
	"strings"

	"github.com/google/uuid"
)

// EmojiData represents a parsed emoji entry.
type EmojiData struct {
	CodePoint   string
	Description string
}

// AlfredSnippet represents the structure of an Alfred snippet JSON file.
type AlfredSnippet struct {
	AlfredSnippet struct {
		Snippet string `json:"snippet"`
		UID     string `json:"uid"`
		Name    string `json:"name"`
		Keyword string `json:"keyword"`
	} `json:"alfredsnippet"`
}

var displayEmojis = flag.Bool("emojis", false, "Display actual emojis instead of code points")

// fetchEmojiData downloads the emoji data from the provided URL.
func fetchEmojiData(url string) ([]string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var lines []string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// parseEmojiLine parses a line of emoji data.
func parseEmojiLine(line string) []*EmojiData {
	parts := strings.Split(line, ";")
	if len(parts) < 3 {
		return nil // Not enough parts in the line
	}

	// Extract code points and description, and remove comments
	codePoints := strings.TrimSpace(parts[0])
	descriptionPart := strings.Split(parts[2], "#")[0]
	description := strings.TrimSpace(descriptionPart)

	// Split multiple emojis and descriptions
	codePointsSplit := strings.Split(codePoints, "..")
	descriptionsSplit := strings.Split(description, "..")

	var emojis []*EmojiData
	for i, codePoint := range codePointsSplit {
		if i < len(descriptionsSplit) {
			emojis = append(emojis, &EmojiData{
				CodePoint:   codePoint,
				Description: descriptionsSplit[i],
			})
		}
	}

	return emojis
}

// convertCodePointToEmoji converts a Unicode code point or points to an emoji character.
func convertCodePointToEmoji(codePointStr string) string {
	var emojiRunes []rune

	// Split the code points if there are multiple parts
	codePoints := strings.Fields(codePointStr)
	for _, cp := range codePoints {
		runeValue, err := strconv.ParseInt(cp, 16, 32)
		if err != nil {
			continue // Skip invalid code point
		}
		emojiRunes = append(emojiRunes, rune(runeValue))
	}

	return string(emojiRunes)
}

// generateAlfredSnippetJSON creates a JSON file for an Alfred snippet.
func generateAlfredSnippetJSON(emoji EmojiData, emojiChar string, uid string, filePath string) error {
	snippet := AlfredSnippet{}
	snippet.AlfredSnippet.Snippet = emojiChar
	snippet.AlfredSnippet.UID = uid
	snippet.AlfredSnippet.Name = emoji.Description
	snippet.AlfredSnippet.Keyword = strings.ReplaceAll(emoji.Description, " ", "-")

	jsonData, err := json.Marshal(snippet)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, jsonData, 0644)
}

// generateUID generates a unique identifier.
func generateUID() string {
	return strings.ToUpper(uuid.New().String())
}

// generateInfoPlist creates the info.plist file for the Alfred snippet pack.
func generateInfoPlist(filePath string) error {
	plistContent := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>snippetkeywordprefix</key>
    <string>:</string>
    <key>snippetkeywordsuffix</key>
    <string></string>
</dict>
</plist>`

	return ioutil.WriteFile(filePath, []byte(plistContent), 0644)
}

// zipFiles creates a .zip file from a list of files.
func zipFiles(zipFileName string, files []string) error {
	newZipFile, err := os.Create(zipFileName)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	for _, file := range files {
		err = addFileToZip(zipWriter, file)
		if err != nil {
			return err
		}
	}

	return nil
}

// addFileToZip adds a single file to the zip archive.
func addFileToZip(zipWriter *zip.Writer, fileName string) error {
	fileToZip, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	// Get the file information
	info, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}

	header.Name = fileName
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, fileToZip)
	return err
}

func main() {
	// Create build and dist directories
	os.Mkdir("build", 0755)
	os.Mkdir("dist", 0755)

	flag.Parse()

	url := "https://unicode.org/Public/emoji/15.1/emoji-sequences.txt"
	lines, err := fetchEmojiData(url)
	if err != nil {
		fmt.Printf("Error fetching emoji data: %v\n", err)
		return
	}

	var filesToZip []string // To keep track of all files to be zipped

	for _, line := range lines {
		emojiData := parseEmojiLine(line)
		for _, emoji := range emojiData {
			if !*displayEmojis {
				emojiChar := convertCodePointToEmoji(emoji.CodePoint)
				uid := generateUID()
				fileName := filepath.Join("", fmt.Sprintf("%s [%s].json", emoji.Description, uid))
				err := generateAlfredSnippetJSON(*emoji, emojiChar, uid, fileName)
				if err != nil {
					fmt.Printf("Error generating JSON for %v: %v\n", emoji.Description, err)
				} else {
					filesToZip = append(filesToZip, fileName)
				}
			}
		}
	}

	if !*displayEmojis {
		plistFileName := filepath.Join("", "info.plist")
		err = generateInfoPlist(plistFileName)
		if err != nil {
			fmt.Printf("Error generating info.plist: %v\n", err)
		} else {
			filesToZip = append(filesToZip, plistFileName)
		}

		iconFileName := "icon.png"
		filesToZip = append(filesToZip, iconFileName)

		zipFileName := filepath.Join("dist", "EmojiPack.alfredsnippets")
		err = zipFiles(zipFileName, filesToZip)
		if err != nil {
			fmt.Printf("Error creating .alfredsnippets file: %v\n", err)
		} else {
			fmt.Println("EmojiPack.alfredsnippets file created successfully.")
			os.Remove("*.json") // Clean up build directory
		}
	}
}
