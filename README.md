# alfredmoji

<p align="center">
  <img src="./icon.png" width="300">
</p>

## Overview
`alfredmoji` is a tool written in Go, designed to generate a Unicode emoji snippet pack specifically for Alfred, a productivity application for macOS. This tool allows users to easily access and use a wide range of emojis within Alfred.

## Features
- Generate a comprehensive collection of Unicode emojis.
- Easy integration with Alfred for macOS users.
- Streamline the process of using emojis in daily workflows.

## Requirements
- Alfred application installed on macOS
- Golang (optional, to compile from source)

## Installation
1. Download the latest 'alfredmoji.alfredsnippets' from the Release page
1. Double-click the downloaded snippet pack to open it in Alred.

## Compilation from Source and Usage
To use `alfredmoji`, follow these steps:
1. Ensure you have Go installed on your system.
1. Clone the repository or download the source code.
1. Navigate to the `src` directory.
1. Run the program using the command `go run main.go` or build & run it with `go build .; ./alfredmoji`
1. You'll find a generated Emoji pack in `./dist/`

## License
`alfredmoji` is distributed under the GNU General Public License. For more information, see the LICENSE/COPYING file in the source code.

## Contributing
Contributions to `alfredmoji` are welcome. Please submit pull requests or issues through the project's repository.
