package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// EmojiData represents a parsed emoji entry.
type EmojiData struct {
	CodePoint   string
	Description string
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

func main() {
	flag.Parse()

	url := "https://unicode.org/Public/emoji/15.1/emoji-sequences.txt"
	lines, err := fetchEmojiData(url)
	if err != nil {
		fmt.Printf("Error fetching emoji data: %v\n", err)
		return
	}

	for _, line := range lines {
		emojiData := parseEmojiLine(line)
		for _, emoji := range emojiData {
			display := emoji.CodePoint
			if *displayEmojis {
				display = convertCodePointToEmoji(emoji.CodePoint)
			}
			fmt.Printf("Emoji: %v, Description: %s\n", display, emoji.Description)
		}
	}
}

