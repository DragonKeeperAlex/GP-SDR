package app

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const radioReferenceKeychainService = "com.gpsdr.radioreference"

type RadioReferenceCredentialUpdate struct {
	Username string `json:"username"`
	Password string `json:"password"`
	AppKey   string `json:"appKey"`
}

func loadRadioReferenceCredentials() RadioReferenceCredentialUpdate {
	credentials := RadioReferenceCredentialUpdate{
		Username: firstEnvironment("GPSDR_RR_USERNAME"),
		Password: firstEnvironment("GPSDR_RR_PASSWORD"),
		AppKey:   firstEnvironment("GPSDR_RR_APP_KEY"),
	}
	if runtime.GOOS != "darwin" {
		return credentials
	}
	if credentials.Username == "" {
		credentials.Username, _ = readKeychainValue("username")
	}
	if credentials.Password == "" {
		credentials.Password, _ = readKeychainValue("password")
	}
	if credentials.AppKey == "" {
		credentials.AppKey, _ = readKeychainValue("app-key")
	}
	return credentials
}

func saveRadioReferenceCredentials(credentials RadioReferenceCredentialUpdate) error {
	credentials.Username = strings.TrimSpace(credentials.Username)
	credentials.AppKey = strings.TrimSpace(credentials.AppKey)
	if credentials.Username == "" || credentials.Password == "" || credentials.AppKey == "" {
		return errors.New("username, password, and approved application key are required")
	}
	if runtime.GOOS != "darwin" {
		return errors.New("secure in-app credential storage currently requires macOS; use the GPSDR_RR_USERNAME, GPSDR_RR_PASSWORD, and GPSDR_RR_APP_KEY environment variables on this platform")
	}
	for account, value := range map[string]string{
		"username": credentials.Username,
		"password": credentials.Password,
		"app-key":  credentials.AppKey,
	} {
		command := exec.Command("/usr/bin/security", "add-generic-password", "-U", "-s", radioReferenceKeychainService, "-a", account, "-w", value)
		if output, err := command.CombinedOutput(); err != nil {
			return fmt.Errorf("could not save RadioReference credentials to Keychain: %s", commandMessage(output, err))
		}
	}
	return nil
}

func clearRadioReferenceCredentials() error {
	if runtime.GOOS != "darwin" {
		return errors.New("secure in-app credential storage currently requires macOS")
	}
	for _, account := range []string{"username", "password", "app-key"} {
		command := exec.Command("/usr/bin/security", "delete-generic-password", "-s", radioReferenceKeychainService, "-a", account)
		if output, err := command.CombinedOutput(); err != nil && !strings.Contains(strings.ToLower(string(output)), "could not be found") {
			return fmt.Errorf("could not clear RadioReference credentials from Keychain: %s", commandMessage(output, err))
		}
	}
	return nil
}

func readKeychainValue(account string) (string, error) {
	command := exec.Command("/usr/bin/security", "find-generic-password", "-s", radioReferenceKeychainService, "-a", account, "-w")
	output, err := command.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(output), "\r\n"), nil
}

func commandMessage(output []byte, err error) string {
	message := strings.TrimSpace(string(output))
	if message != "" {
		return message
	}
	return err.Error()
}

func radioReferenceCredentialSource(credentials RadioReferenceCredentialUpdate) string {
	if os.Getenv("GPSDR_RR_USERNAME") != "" || os.Getenv("GPSDR_RR_PASSWORD") != "" || os.Getenv("GPSDR_RR_APP_KEY") != "" {
		return "Environment"
	}
	if runtime.GOOS == "darwin" && (credentials.Username != "" || credentials.Password != "" || credentials.AppKey != "") {
		return "Mac Keychain"
	}
	return "Not configured"
}

func maskedAccount(username string) string {
	username = strings.TrimSpace(username)
	if len(username) <= 2 {
		return strings.Repeat("•", len(username))
	}
	return username[:1] + strings.Repeat("•", len(username)-2) + username[len(username)-1:]
}
