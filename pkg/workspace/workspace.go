// Copyright 2016-2018, Pulumi Corporation.  All rights reserved.

package workspace

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/pulumi/pulumi/pkg/resource/config"
	"github.com/pulumi/pulumi/pkg/tokens"
	"github.com/pulumi/pulumi/pkg/util/contract"
)

// W offers functionality for interacting with Pulumi workspaces.
type W interface {
	Settings() *Settings // returns a mutable pointer to the optional workspace settings info.
	Save() error         // saves any modifications to the workspace.
}

type projectWorkspace struct {
	name     tokens.PackageName // the package this workspace is associated with.
	project  string             // the path to the Pulumi.[yaml|json] file for this project.
	settings *Settings          // settings for this workspace.
}

var cache = make(map[string]W)
var cacheMutex sync.RWMutex

func loadFromCache(key string) (W, bool) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	w, ok := cache[key]
	return w, ok
}

func upsertIntoCache(key string, w W) {
	contract.Require(w != nil, "w")

	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	cache[key] = w
}

// New creates a new workspace using the current working directory.
func New() (W, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return NewFrom(cwd)
}

// NewFrom creates a new Pulumi workspace in the given directory. Requires a Pulumi.yaml file be present in the
// folder hierarchy between dir and the .pulumi folder.
func NewFrom(dir string) (W, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	dir = absDir

	if w, ok := loadFromCache(dir); ok {
		return w, nil
	}

	path, err := DetectProjectPathFrom(dir)
	if err != nil {
		return nil, err
	} else if path == "" {
		return nil, errors.New("no Pulumi.yaml project file found")
	}

	proj, err := LoadProject(path)
	if err != nil {
		return nil, err
	}

	w := &projectWorkspace{
		name:    proj.Name,
		project: path,
	}

	err = w.readSettings()
	if err != nil {
		return nil, err
	}

	if w.settings.ConfigDeprecated == nil {
		w.settings.ConfigDeprecated = make(map[tokens.QName]config.Map)
	}

	upsertIntoCache(dir, w)
	return w, nil
}

func (pw *projectWorkspace) Settings() *Settings {
	return pw.settings
}

func (pw *projectWorkspace) Save() error {
	// let's remove all the empty entries from the config array
	for k, v := range pw.settings.ConfigDeprecated {
		if len(v) == 0 {
			delete(pw.settings.ConfigDeprecated, k)
		}
	}

	settingsFile := pw.settingsPath()

	// If the settings file is empty, don't write an new one, and delete the old one if present. Since we put workspaces
	// under ~/.pulumi/workspaces, cleaning them out when possible prevents us from littering a bunch of files in the
	// home directory.
	if pw.settings.IsEmpty() {
		err := os.Remove(settingsFile)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}

	err := os.MkdirAll(filepath.Dir(settingsFile), 0700)
	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(pw.settings, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(settingsFile, b, 0600)
}

func (pw *projectWorkspace) readSettings() error {
	settingsPath := pw.settingsPath()

	b, err := ioutil.ReadFile(settingsPath)
	if err != nil && os.IsNotExist(err) {
		// not an error to not have an existing settings file.
		pw.settings = &Settings{}
		return nil
	} else if err != nil {
		return err
	}

	var settings Settings

	err = json.Unmarshal(b, &settings)
	if err != nil {
		return err
	}

	pw.settings = &settings
	return nil
}

func (pw *projectWorkspace) settingsPath() string {
	user, err := user.Current()
	contract.AssertNoErrorf(err, "could not get current user")

	uniqueFileName := string(pw.name) + "-" + sha1HexString(pw.project) + "-" + WorkspaceFile
	return filepath.Join(user.HomeDir, BookkeepingDir, WorkspaceDir, uniqueFileName)
}

// sha1HexString returns a hex string of the sha1 hash of value.
func sha1HexString(value string) string {
	h := sha1.New()
	_, err := h.Write([]byte(value))
	contract.AssertNoError(err)
	return hex.EncodeToString(h.Sum(nil))
}

// qnameFileName takes a qname and cleans it for use as a filename (by replacing tokens.QNameDelimter with a dash)
func qnameFileName(nm tokens.QName) string {
	return strings.Replace(string(nm), tokens.QNameDelimiter, "-", -1)
}
