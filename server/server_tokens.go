package server

import (
	"bufio"
	rand "crypto/rand"
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// AddToken creates a new token for the service/instance if it does not yet exist
func (l *logServer) AddToken(service, instance string) (string, error) {
	l.Lock()
	defer l.Unlock()

	// Clean the key
	key := getCleanKey(service, instance)

	// Verify key existence
	if _, ok := l.tokens[key]; ok {
		return "", fmt.Errorf("AddToken: token for %s already exists", key)
	}

	// Create a random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("AddToken: could not generate a random token: %s", err.Error())
	}
	token := fmt.Sprintf("%x", sha256.Sum256(tokenBytes))

	// Write the token database to file
	if err := l.writeTokenToFile(key, token); err != nil {
		return "", fmt.Errorf("AddToken: could not write token to file: %s", err.Error())
	}

	// Assign token to the key
	l.tokens[key] = token
	l.stats[key] = &Statistic{
		Service:  service,
		Instance: instance,
	}

	return token, nil
}

// GetTokens returns LogServer's tokens
func (l *logServer) GetTokens() map[string]string {
	l.Lock()
	l.Unlock()

	copyTokens := map[string]string{}
	for key, token := range l.tokens {
		copyTokens[key] = token
	}

	return copyTokens
}

// RemoveTokens removes all the authentication tokens of a service
func (l *logServer) RemoveTokens(service string) error {
	l.Lock()
	defer l.Unlock()

	// Identify all the keys belonging to a service
	keys := []string{}
	for key := range l.tokens {
		if parts := strings.Split(key, "/"); parts[0] == service {
			keys = append(keys, key)
		}
	}

	// Remove keys one by one
	for _, key := range keys {
		parts := strings.Split(key, "/")
		if err := l.RemoveToken(parts[0], parts[1], false); err != nil {
			return fmt.Errorf("RemoveTokens: could not remove token for key '%s': %s", key, err.Error())
		}
	}

	return nil
}

// RemoveToken removes an authentication token
func (l *logServer) RemoveToken(service, instance string, lock bool) error {
	if lock {
		l.Lock()
		defer l.Unlock()
	}

	// Clean the key
	key := getCleanKey(service, instance)

	// Check that the key exists
	if _, ok := l.tokens[key]; !ok {
		return fmt.Errorf("RemoveToken: no such service/instance")
	}

	// Remove the token from file
	if err := l.removeTokenFromFile(key, false); err != nil {
		return fmt.Errorf("RemoveToken: could not remove token for %s: %s", key, err.Error())
	}

	// Remove from memory
	delete(l.tokens, key)

	return nil
}

// writeTokenToFile writes a tokens to file
func (l *logServer) writeTokenToFile(key, token string) error {

	// Make sure file is writeable
	if err := fileExists(l.tokenPath); err != nil {
		return fmt.Errorf("writeTokenToFile: could not create tokens.db: %s", err.Error())
	}

	// Write to file
	f, err := os.OpenFile(l.tokenPath, os.O_WRONLY|os.O_APPEND, 0600)
	if err == nil {
		if _, err = f.WriteString(fmt.Sprintf("%s\t%s\n", key, token)); err != nil {
			return fmt.Errorf("writeTokenToFile: could not write token to file: %s", err.Error())
		}
	} else {
		return fmt.Errorf("writeTokenToFile: could not open file: %s", err.Error())
	}

	return f.Close()

}

// removeTokenFromFile removes a single token from the tokens.db
func (l *logServer) removeTokenFromFile(key string, lock bool) error {
	if lock {
		l.Lock()
		defer l.Unlock()
	}

	// Make sure file exists
	if err := fileExists(l.tokenPath); err != nil {
		return fmt.Errorf("removeTokenFromFile: could not create tokens database: %s", err.Error())
	}

	// Open file for reading
	f, err := os.OpenFile(l.tokenPath, os.O_RDWR, 600)
	if err != nil {
		return fmt.Errorf("removeTokenFromFile: could not open token database for reading: %s", err.Error())
	}

	// Read all except for the key
	fileScanner := bufio.NewScanner(f)
	tokens := []string{}
	for fileScanner.Scan() {
		line := fileScanner.Text()

		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			continue
		}
		keyParts := strings.Split(parts[0], "/")
		if len(keyParts) != 2 {
			continue
		}

		if parts[0] != key {
			tokens = append(tokens, line)
		}
	}

	if err := f.Close(); err != nil {
		return err
	}

	tokens = append(tokens, "\n")

	// Revwrite tokens.db
	if err := ioutil.WriteFile(l.tokenPath, []byte(strings.Join(tokens, "\n")), 0600); err != nil {
		return fmt.Errorf("removeTokenFromFile: could not rewrite token database: %s", err.Error())
	}

	return nil
}

// loadTokensFromDisk loads all the tokens from disk to memory
func (l *logServer) loadTokensFromDisk() error {
	l.Lock()
	defer l.Unlock()

	// Make sure file exists
	if err := fileExists(l.tokenPath); err != nil {
		return fmt.Errorf("loadTokensFromDisk: could not create tokens.db: %s", err.Error())
	}

	// Open file for reading
	f, err := os.OpenFile(l.tokenPath, os.O_RDONLY, 0600)
	if err != nil {
		return fmt.Errorf("loadTokensFromDisk: could not open token file for reading: %s", err.Error())
	}

	// Read line by line and add to the in-memory db
	fileScanner := bufio.NewScanner(f)
	for fileScanner.Scan() {
		line := fileScanner.Text()
		parts := strings.Split(line, "\t")
		if len(parts) != 2 {
			continue
		}
		keyParts := strings.Split(parts[0], "/")
		if len(keyParts) != 2 {
			continue
		}
		l.tokens[parts[0]] = parts[1]
	}

	return f.Close()
}
