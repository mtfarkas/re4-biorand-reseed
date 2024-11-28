package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/mtfarkas/re4-biorand-reseed/json_ex"
)

var seedRunes = []rune("0123456789")

const SEPARATOR = "================================"
const RANDO_PROFILE_ID = 7 // 7rayD's Balanced Combat Randomizer

type Config struct {
	RE4InstallPath string
	BiorandToken   string
}

type BiorandProfile struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	ConfigId    int                    `json:"configId"`
	Config      map[string]interface{} `json:"config"`
}

type GenerateRequest struct {
	ProfileID int                    `json:"profileId"`
	Seed      string                 `json:"seed"`
	Config    map[string]interface{} `json:"config"`
}

type GenerateResponse struct {
	ID      int    `json:"id"`
	Version string `json:"version"`
	Status  int    `json:"status"`
}

type QueryGenerationResponse struct {
	Status      int    `json:"status"`
	DownloadUrl string `json:"downloadUrl"`
}

func getAuthenticatedHttpRequest(url string, method string, token string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating HTTP request; %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	return req, nil
}

func generateSeed() string {
	b := make([]rune, 6)
	for i := range b {
		b[i] = seedRunes[rand.Intn(len(seedRunes))]
	}
	return string(b)
}

func getConfiguration() (*Config, error) {
	jsonFile, err := os.Open("reseed-config.json")
	if err != nil {
		return nil, fmt.Errorf("error reading config file; %w", err)
	}
	defer jsonFile.Close()
	allBytes, _ := io.ReadAll(jsonFile)

	config, err := json_ex.GenericUnmarshal[Config](allBytes)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling config file; %w", err)
	}

	if len(config.BiorandToken) < 1 {
		return nil, fmt.Errorf("the Biorand token can't be empty")
	}
	if len(config.RE4InstallPath) < 1 {
		return nil, fmt.Errorf("RE4 install path can't be empty")
	}

	return &config, nil
}

func getBiorandProfileConfiguration(profileId int, biorandToken string) (*BiorandProfile, error) {
	client := http.Client{}

	req, err := getAuthenticatedHttpRequest("https://api-re4r.biorand.net/profile", "GET", biorandToken, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating Profile request; %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling Profile API; %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading Profile API response; %w", err)
		}
		profiles, err := json_ex.GenericUnmarshal[[]BiorandProfile](bodyBytes)
		if err != nil {
			return nil, fmt.Errorf("error unmarshaling Profile API response; %w", err)
		}
		idx := slices.IndexFunc(profiles, func(p BiorandProfile) bool { return p.ID == profileId })
		if idx < 0 {
			return nil, fmt.Errorf("couldn't find profile with ID %d in Profile API response. Make sure you bookmark it with your account", profileId)
		}
		return &profiles[idx], nil
	} else {
		return nil, fmt.Errorf("response from Profile API doesn't indicate success; %d", resp.StatusCode)
	}
}

func generateBiorandSeed(seed string, profile *BiorandProfile, biorandToken string) (*GenerateResponse, error) {
	client := http.Client{}

	reqBody := GenerateRequest{
		ProfileID: profile.ID,
		Seed:      seed,
		Config:    profile.Config,
	}
	reqBodyBytes, _ := json.Marshal(reqBody)

	req, err := getAuthenticatedHttpRequest("https://api-re4r.biorand.net/rando/generate", "POST", biorandToken, bytes.NewBuffer(reqBodyBytes))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, fmt.Errorf("error creating Generate request; %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling Generate API; %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response from Generate API doesn't indicate success; %d", resp.StatusCode)
	}
	generateResponseBytes, _ := io.ReadAll(resp.Body)
	generateResponse, err := json_ex.GenericUnmarshal[GenerateResponse](generateResponseBytes)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling Generate API response; %w", err)
	}

	return &generateResponse, nil
}

func queryBiorandSeedDownloadLink(generateStartResponse *GenerateResponse, biorandToken string) (*QueryGenerationResponse, error) {
	client := http.Client{}
	req, err := getAuthenticatedHttpRequest(fmt.Sprintf("https://api-re4r.biorand.net/rando/%d", generateStartResponse.ID), "GET", biorandToken, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating Query Generate request; %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling Query Generate API; %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response from Query Generate API doesn't indicate success; %d", resp.StatusCode)
	}
	queryResponseBytes, _ := io.ReadAll(resp.Body)
	queryResponse, err := json_ex.GenericUnmarshal[QueryGenerationResponse](queryResponseBytes)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling Query Generate API response; %w", err)
	}
	return &queryResponse, nil
}

func downloadSeedZip(seed string, downloadUrl string) (string, error) {
	client := http.Client{}

	resp, err := client.Get(downloadUrl)
	if err != nil {
		return "", fmt.Errorf("error getting seed zip response; %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download file response doesn't indicate success: %d", resp.StatusCode)
	}

	fileName := fmt.Sprintf("biorand-re4r-%s.zip", seed)
	contentDispositionHeader := resp.Header.Get("Content-Disposition")
	if len(contentDispositionHeader) > 0 {
		_, params, err := mime.ParseMediaType(contentDispositionHeader)
		if err == nil {
			fileName = params["filename"]
		}
	}

	downloadDir := path.Join("biorand-seeds", seed)
	err = os.MkdirAll(downloadDir, os.FileMode(int(0777)))
	if err != nil {
		return "", fmt.Errorf("failed to create biorand seed folder; %w", err)
	}

	fullFilePath := path.Join(downloadDir, fileName)

	out, err := os.Create(fullFilePath)
	if err != nil {
		return "", fmt.Errorf("error creating file to download to; %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", fmt.Errorf("error saving seed file; %w", err)
	}

	return fullFilePath, nil
}

func unzipArchiveToDestination(zipFile string, dest string) error {
	r, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("error opening zip file %s; %w", zipFile, err)
	}
	defer r.Close()

	for _, f := range r.File {
		fmt.Printf("Unzipping %s...\n", f.Name)

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("error reading file from zip; %w", err)
		}
		defer rc.Close()

		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
		} else {
			var fdir string
			if lastIndex := strings.LastIndex(fpath, string(os.PathSeparator)); lastIndex > -1 {
				fdir = fpath[:lastIndex]
			}

			err = os.MkdirAll(fdir, f.Mode())
			if err != nil {
				return fmt.Errorf("error creating directories while unzipping; %w", err)
			}
			f, err := os.OpenFile(
				fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = io.Copy(f, rc)
			if err != nil {
				return fmt.Errorf("error writing unzipped file; %w", err)
			}
		}
	}
	return nil
}

func waitForKeyPress() {
	fmt.Println("Press any key to continue...")
	bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func main() {
	defer waitForKeyPress()

	fmt.Println("Generating new seed...")

	config, err := getConfiguration()
	if err != nil {
		fmt.Printf("Error getting configuration: %v\n", err)
		return
	}

	fmt.Printf("RE4 path: %s\n", config.RE4InstallPath)
	fmt.Printf("Profile ID: %d\n", RANDO_PROFILE_ID)
	fmt.Println()

	fmt.Println("Getting randomizer profile...")
	profile, err := getBiorandProfileConfiguration(RANDO_PROFILE_ID, config.BiorandToken)
	if err != nil {
		fmt.Printf("Error getting profile information: %v\n", err)
		return
	}

	fmt.Println("Profile info downloaded.")
	fmt.Printf("Profile name: %s\n", profile.Name)
	fmt.Printf("Profile description: %s\n", profile.Description)
	fmt.Println()

	seed := generateSeed()
	fmt.Printf("Generated the following seed: %s", seed)
	fmt.Println()

	fmt.Println("Continue? (y/n)")
	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	if !strings.EqualFold(strings.Trim(text, "\r\n"), "y") {
		fmt.Println("Reseeding aborted.")
		return
	}

	fmt.Println()
	fmt.Println("Generating new seed on Biorand...")
	seedResponse, err := generateBiorandSeed(seed, profile, config.BiorandToken)
	if err != nil {
		fmt.Printf("Error generating Biorand seed: %v\n", err)
		return
	}

	downloadLink := ""
	errorCount := 0
	attemptCount := 0
	for {
		attemptCount += 1

		if attemptCount >= 180 {
			fmt.Println("Seed generation timed out. Aborting.")
			return
		}
		queryResponse, err := queryBiorandSeedDownloadLink(seedResponse, config.BiorandToken)
		if err != nil {
			errorCount += 1
			fmt.Printf("Error querying Biorand API (%d/3): %v\n", errorCount, err)

			if errorCount > 3 {
				fmt.Printf("Error count treshold reached. Aborting.")
				return
			}
		}
		errorCount = 0

		if queryResponse.Status == 1 {
			fmt.Println("Seed is queued for generation.")
		} else if queryResponse.Status == 2 {
			fmt.Println("Seed is being generated.")
		} else if queryResponse.Status == 3 {
			fmt.Println("Seed is done generating.")
			downloadLink = queryResponse.DownloadUrl
			break
		} else {
			fmt.Println("Seed status unknown; Aborting.")
			return
		}

		time.Sleep(1 * time.Second)
	}

	fmt.Println()
	fmt.Println("Downloading seed zip...")
	zipPath, err := downloadSeedZip(seed, downloadLink)
	if err != nil {
		fmt.Printf("Error downloading seed zip: %v\n", err)
		return
	}

	fmt.Printf("Seed zip successfully downloaded to %s\n", zipPath)
	fmt.Println()

	fmt.Printf("Unzipping seed zip to %s...\n", config.RE4InstallPath)
	err = unzipArchiveToDestination(zipPath, config.RE4InstallPath)
	if err != nil {
		fmt.Printf("Failed to unzip seed: %v\n", err)
	}

	fmt.Println("Reseeding completed. Enjoy!")
	fmt.Println()
}
