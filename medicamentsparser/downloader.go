// Package medicamentsparser provides functionality for downloading and parsing medicament data from external sources.
package medicamentsparser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"unicode/utf8"

	"github.com/giygas/medicaments-api/logging"
	"golang.org/x/text/encoding/charmap"
)

func downloadAndParseFile(filepath string, url string) error {

	filepath = "files/" + filepath + ".txt"
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer func() {
		if err = response.Body.Close(); err != nil {
			logging.Warn("Failed to close response body", "error", err)
		}
	}()

	// As there are some files in iso-8859-1 and some in utf8, read the content first
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	// Check if it's valid UTF-8
	var reader io.Reader
	if utf8.Valid(bodyBytes) {
		// Already UTF-8, use as-is
		reader = bytes.NewReader(bodyBytes)
	} else {
		// Not UTF-8, decode from ISO-8859-1
		reader = charmap.ISO8859_1.NewDecoder().Reader(bytes.NewReader(bodyBytes))
	}

	outFile, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filepath, err)
	}
	defer func() {
		if err = outFile.Close(); err != nil {
			logging.Warn("Failed to close output file", "error", err)
		}
	}()

	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		_, err = fmt.Fprintln(outFile, scanner.Text())
		if err != nil {
			return fmt.Errorf("failed to write to file %s: %w", filepath, err)
		}
	}

	logging.Debug(fmt.Sprintf("%s downloaded and parsed without errors", filepath))
	return nil
}

// Download all files concurrently
func downloadAndParseAll() error {

	//Files to download
	var files = map[string]string{
		"Specialites":   "https://base-donnees-publique.medicaments.gouv.fr/download/file/CIS_bdpm.txt",
		"Presentations": "https://base-donnees-publique.medicaments.gouv.fr/download/file/CIS_CIP_bdpm.txt",
		"Compositions":  "https://base-donnees-publique.medicaments.gouv.fr/download/file/CIS_COMPO_bdpm.txt",
		"Generiques":    "https://base-donnees-publique.medicaments.gouv.fr/download/file/CIS_GENER_bdpm.txt",
		"Conditions":    "https://base-donnees-publique.medicaments.gouv.fr/download/file/CIS_CPD_bdpm.txt",
	}

	//Create the files directory if it doesn't exists
	filePath := filepath.Join(".", "files")
	err := os.MkdirAll(filePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create files directory: %w", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	for fileName, url := range files {
		wg.Add(1)

		go func(file string, url string) {
			defer wg.Done()
			if err := downloadAndParseFile(file, url); err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
			}
		}(fileName, url)

	}
	wg.Wait()

	if len(errors) > 0 {
		logging.Error("Download errors occurred", "errors", errors)
		return fmt.Errorf("download errors: %v", errors)
	}

	return nil
}
