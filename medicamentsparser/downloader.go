// Package medicamentsparser provides functionality for downloading and parsing medicament data from external sources.
package medicamentsparser

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/giygas/medicamentsfr/logging"
	"golang.org/x/text/encoding/charmap"
)

func downloadAndParseFile(filepath string, url string) error {

	filepath = "files/" + filepath + ".txt"
	response, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer response.Body.Close()

	outFile, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", filepath, err)
	}

	defer outFile.Close()
	reader := charmap.Windows1252.NewDecoder().Reader(response.Body)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		_, err = fmt.Fprintln(outFile, scanner.Text())
		if err != nil {
			return fmt.Errorf("failed to write to file %s: %w", filepath, err)
		}
	}
	fmt.Println("Downloaded: " + filepath)
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
	filePath = filepath.Join(".", "src")
	err = os.MkdirAll(filePath, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create src directory: %w", err)
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
		logger := logging.SetupLogger("logs")
		logger.Error("Download errors occurred", "errors", errors)
		return fmt.Errorf("download errors: %v", errors)
	}

	return nil
}
