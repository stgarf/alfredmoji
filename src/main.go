// alfredmoji, generate unicode emoji snippetpack for Alfred
// Copyright (C) 2024  Steven Garf

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"archive/zip"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"

	"strings"

	"github.com/google/uuid"
)

// EmojiData represents a parsed emoji entry.
type EmojiData struct {
	Emoji       string
	Description string
	Subgroup    string
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

// displayEmojis is a flag to display emojis instead of generating an Alfred snippet pack.
var displayEmojis = flag.Bool("emojis", false, "Display emojis instead of generating Alfred snippet pack")

// unicodeVersion is a flag to specify the Unicode version to use.
var unicodeVersion = flag.String("version", "15.1", "Unicode version to use (default: 15.1)")

// unicodeEmojiURL is the URL to download the emoji data from.
var unicodeEmojiURL = "https://unicode.org/Public/emoji/%s/emoji-test.txt"

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
    <string>:</string>
</dict>
</plist>`

	return os.WriteFile(filePath, []byte(plistContent), 0644)
}

// fetchEmojiData downloads the emoji data from the provided URL if it does not exist locally.
func fetchEmojiData(url string) ([]string, error) {
	// Get filename from URL
	_, filename := path.Split(url)

	// Hold lines from file
	var lines []string

	if _, err := os.Stat(filename); err == nil {
		// If file exists
		fmt.Printf("Using existing file: %s\n", filename)
		file, err := os.Open(filename)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
	} else {
		// File does not exist, download
		fmt.Printf("Downloading file: %s\n", url)
		file, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer file.Body.Close()
		scanner := bufio.NewScanner(file.Body)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		// Write file to disk
		os.WriteFile(filename, []byte(strings.Join(lines, "\n")), 0644)
	}

	return lines, nil
}

// extractDescriptionAndEmoji extracts the emoji and description from a line of emoji data.
func extractDescriptionAndEmoji(input string) (string, string) {
	// regex for: '# 😀 E1.0 grinning face'
	re := regexp.MustCompile(`^\s+(.+) E\d+\.\d+ (.+)`)
	matches := re.FindStringSubmatch(input)

	// if we got the right number of matches, return the emoji and description
	if len(matches) == 3 {
		emoji := strings.TrimSpace(matches[1])

		// Remove "junk" characters from description
		description := strings.TrimSpace(strings.ReplaceAll(matches[2], " ", "-"))
		description = strings.ReplaceAll(description, ",", "")
		description = strings.ReplaceAll(description, ":", "")
		description = strings.ReplaceAll(description, "’", "")
		description = strings.ReplaceAll(description, "‘", "")
		description = strings.ReplaceAll(description, "“", "")
		description = strings.ReplaceAll(description, "”", "")
		return emoji, description
	}

	// otherwise, return empty strings
	return "", ""
}

// parseEmojiLine parses a line of emoji data and updates the current subgroup if needed.
func parseEmojiLine(line string, currentSubgroup *string) []*EmojiData {
	if strings.HasPrefix(line, "# subgroup:") {
		// Update the current subgroup
		*currentSubgroup = strings.TrimSpace(strings.TrimPrefix(line, "# subgroup:"))
		return nil
	}

	// Skip lines that are not emoji definitions
	// This skips lines that start with "#", lines that contain "minimally-qualified", "unqualified", or "component"
	// This skips lines that do not contain a ";", or "#"
	if !strings.Contains(line, ";") || !strings.Contains(line, "#") || strings.HasPrefix(line, "#") || strings.Contains(line, "minimally-qualified") || strings.Contains(line, "unqualified") || strings.Contains(line, "component") {
		return nil
	}

	// 1F600		; fully-qualified		# 😀 E1.0 grinning face
	// Extract code points (ignored), qualification (ignored), and description
	parts := strings.SplitN(line, ";", 2)
	// Get the interesting bits
	interestingBits := strings.Split(parts[1], "#")[1]
	emoji, description := extractDescriptionAndEmoji(interestingBits)

	var emojis []*EmojiData
	// Add emoji to list
	emojis = append(emojis, &EmojiData{
		Emoji:       emoji,
		Description: description,
		Subgroup:    *currentSubgroup,
	})

	return emojis
}

// generateAlfredSnippetJSON creates a JSON file for an Alfred snippet.
func generateAlfredSnippetJSON(emoji EmojiData, emojiChar string, uid string, filePath string) error {
	// Create snippet
	snippet := AlfredSnippet{}
	// Set values
	snippet.AlfredSnippet.Snippet = emojiChar
	snippet.AlfredSnippet.UID = uid
	snippet.AlfredSnippet.Name = fmt.Sprintf("(%s) %s", emoji.Subgroup, emoji.Description)
	snippet.AlfredSnippet.Keyword = emoji.Description

	// Marshal snippet to JSON
	jsonData, err := json.Marshal(snippet)
	if err != nil {
		return err
	}

	// Write JSON to file
	return os.WriteFile(filePath, jsonData, 0644)
}

// zipFiles creates a .zip file from a list of files.
func zipFiles(filename string, files []string) error {
	// Create the new zip file
	newZipFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	// Close the file at the end of this function
	defer newZipFile.Close()

	// Create a zip writer
	zipWriter := zip.NewWriter(newZipFile)
	// Close the zip writer at the end of this function
	defer zipWriter.Close()

	// Add files to zip
	for _, file := range files {
		if err = addFileToZip(zipWriter, file); err != nil {
			return err
		}
	}
	return nil
}

// addFileToZip adds a single file to the zip archive.
func addFileToZip(zipWriter *zip.Writer, filename string) error {
	// Open the file to be zipped
	fileToZip, err := os.Open(filename)
	if err != nil {
		return err
	}
	// Close the file at the end of this function
	defer fileToZip.Close()

	// Get the name of the file without the "build/" prefix
	_, name := filepath.Split(filename)

	// Create a new writer for the file
	zipFileWriter, err := zipWriter.Create(name)
	if err != nil {
		return err
	}

	// Copy the file to the zip writer
	_, err = io.Copy(zipFileWriter, fileToZip)

	// Return any errors
	return err
}

func main() {
	// Create build and dist directories
	os.Mkdir("build", 0755)
	os.Mkdir("dist", 0755)

	// Parse cmd line flags
	flag.Parse()

	// Get emojis from unicode.org
	url := fmt.Sprintf(unicodeEmojiURL, *unicodeVersion)

	// Get lines from file
	lines, err := fetchEmojiData(url)
	if err != nil {
		fmt.Printf("Error fetching emoji data: %v\n", err)
		return
	}

	// To keep track of all files to be zipped
	var filesToZip []string

	// Keep track of current subgroup
	var currentSubgroup *string

	// Loop through each line
	for _, line := range lines {
		// If the line starts with "# subgroup:", update the current subgroup, else parse the line
		if strings.HasPrefix(line, "# subgroup:") {
			// Clean up subgroup name
			subgroup := strings.TrimSpace(strings.TrimPrefix(line, "# subgroup:"))
			subgroup = strings.ReplaceAll(subgroup, "&", "and")
			subgroup = strings.ReplaceAll(subgroup, " ", "-")
			// Set current subgroup
			currentSubgroup = &subgroup
			// Skip to next line
			continue
		} else if currentSubgroup != nil {
			// Parse the line and get the emoji data
			emojiData := parseEmojiLine(line, currentSubgroup)

			// If we got emoji data, generate Alfred snippet JSON
			for _, emoji := range emojiData {
				// If flag is set to display emojis, display them instead of generating snippet pack
				if !*displayEmojis {
					// Generate a unique identifier
					uid := generateUID()

					// Generate JSON filename
					fileName := filepath.Join("build", fmt.Sprintf("%s [%s].json", emoji.Description, uid))
					// Generate JSON file and add to list of files to zip
					err := generateAlfredSnippetJSON(*emoji, emoji.Emoji, uid, fileName)
					if err != nil {
						fmt.Printf("Error generating JSON for %v: %v\n", emoji.Description, err)
					} else {
						filesToZip = append(filesToZip, fileName)
					}
				} else {
					// Display emoji
					emojiChar := emoji.Emoji
					fmt.Printf("%s: %s\n", emojiChar, emoji.Description)
				}
			}
		}
	}

	if !*displayEmojis {
		// Generate info.plist file and add to list of files to zip
		plistFileName := filepath.Join("build", "info.plist")
		err = generateInfoPlist(plistFileName)
		if err != nil {
			fmt.Printf("Error generating info.plist: %v\n", err)
		} else {
			filesToZip = append(filesToZip, plistFileName)
		}

		// Add icon to list of files to zip
		iconFileName := "../icon.png"
		filesToZip = append(filesToZip, iconFileName)

		// Generate zip file name
		packName := fmt.Sprintf("alfredmoji-%s.alfredsnippets", *unicodeVersion)
		zipFileName := filepath.Join("dist", packName)
		// Zip files
		err = zipFiles(zipFileName, filesToZip)
		if err != nil {
			fmt.Printf("Error creating .alfredsnippets file: %v\n", err)
		} else {
			fmt.Println("alfredmoji.alfredsnippets file created successfully.")
			// Clean up build directory
			os.RemoveAll("build")
		}
	}
}
