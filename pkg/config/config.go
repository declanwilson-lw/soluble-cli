// Copyright 2020 Soluble Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/soluble-ai/go-jnode"
	"github.com/soluble-ai/soluble-cli/pkg/log"
	"github.com/soluble-ai/soluble-cli/pkg/util"
	"gopkg.in/yaml.v3"
)

var GlobalConfig = &struct {
	Profiles       map[string]*ProfileT
	CurrentProfile string
	ModelLocations []string
}{}

// Config points to the current profile
var (
	Config             = &ProfileT{}
	ConfigFile         string
	ConfigDir          string
	configFileRead     string
	migrationAvailable bool
)

const Redacted = "*** redacted ***"

type ProfileT struct {
	ProfileName  string `json:"-"`
	APIServer    string
	APIToken     string
	TLSNoVerify  bool
	Organization string
	Email        string
}

func SelectProfile(name string) bool {
	result := false
	Config = GlobalConfig.Profiles[name]
	if Config == nil {
		Config = &ProfileT{
			APIServer: "https://api.soluble.cloud",
		}
		GlobalConfig.Profiles[name] = Config
		result = true
	}
	GlobalConfig.CurrentProfile = name
	Config.ProfileName = name
	return result
}

func CopyProfile(sourceName string) error {
	if GlobalConfig.CurrentProfile == "" {
		return fmt.Errorf("no current profile to copy to")
	}
	source := GlobalConfig.Profiles[sourceName]
	if source == nil {
		return fmt.Errorf("no source profile %s to copy from", sourceName)
	}
	copy := *source
	copy.ProfileName = GlobalConfig.CurrentProfile
	GlobalConfig.Profiles[GlobalConfig.CurrentProfile] = &copy
	SelectProfile(GlobalConfig.CurrentProfile)
	return nil
}

func DeleteProfile(name string) bool {
	if GlobalConfig.Profiles[name] == nil {
		log.Warnf("Deleting non-existent profile {warning:%s} has no effect", name)
		return false
	}
	delete(GlobalConfig.Profiles, name)
	log.Infof("Deleted profile {info:%s}", name)
	if len(GlobalConfig.Profiles) == 0 {
		SelectProfile("default")
	} else if name == GlobalConfig.CurrentProfile {
		for p := range GlobalConfig.Profiles {
			SelectProfile(p)
			break
		}
	}
	return true
}

func RenameProfile(name, rename string) error {
	p := GlobalConfig.Profiles[name]
	if p == nil {
		return fmt.Errorf("cannot rename non-existent profile %s", name)
	}
	GlobalConfig.Profiles[rename] = p
	p.ProfileName = rename
	delete(GlobalConfig.Profiles, name)
	if GlobalConfig.CurrentProfile == name {
		GlobalConfig.CurrentProfile = rename
	}
	log.Infof("Renamed profile {info:%s} to {info:%s}", name, rename)
	return nil
}

func (c *ProfileT) String() string {
	cfg := *Config
	if cfg.APIToken != "" {
		cfg.APIToken = Redacted
	}
	var m map[string]interface{}
	dat, _ := json.Marshal(cfg)
	_ = json.Unmarshal(dat, &m)
	m["ConfigFile"] = configFileRead
	s, _ := yaml.Marshal(m)
	return string(s)
}

func (c *ProfileT) GetAppURL() string {
	const httpAPI = "https://api."
	if strings.HasPrefix(c.APIServer, httpAPI) {
		return "https://app." + c.APIServer[len(httpAPI):]
	}
	return "https://app.soluble.cloud"
}

func (c *ProfileT) GetAPIToken() string {
	token := strings.TrimSpace(os.Getenv("SOLUBLE_API_TOKEN"))
	if token != "" {
		return token
	}
	return c.APIToken
}

func (c *ProfileT) AssertAPITokenFromConfig() error {
	if os.Getenv("SOLUBLE_API_TOKEN") != "" {
		return fmt.Errorf("the environment variable SOLUBLE_API_TOKEN is set")
	}
	return nil
}

func (c *ProfileT) GetAPIServer() string {
	server := strings.TrimSpace(os.Getenv(("SOLUBLE_API_SERVER")))
	if server != "" {
		return server
	}
	return c.APIServer
}

func Save() error {
	dir := filepath.Dir(ConfigFile)
	if dir != "" {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			err := os.MkdirAll(dir, 0777)
			if err != nil {
				log.Errorf("Could not create config directory {info:%s}: {danger:%s}", dir, err.Error())
				return err
			}
		}
	}
	if configFileRead != "" && configFileRead != ConfigFile {
		log.Infof("Migrating legacy config file {info:%s} to {info:%s}", configFileRead, ConfigFile)
		err := os.Rename(configFileRead, ConfigFile)
		if err != nil {
			log.Errorf("Could not rename legacy config file {info:%s} to {info:%s}: {danger:%s}",
				configFileRead, ConfigFile, err.Error())
			return err
		}
	}
	log.Infof("Updating {info:%s}\n", ConfigFile)
	file, err := os.OpenFile(ConfigFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err == nil {
		defer file.Close()
		enc := json.NewEncoder(file)
		enc.SetIndent("", "  ")
		err = enc.Encode(GlobalConfig)
	}
	if err != nil {
		log.Errorf("Failed to save {info:%s}: {danger:%s}", ConfigFile, err.Error())
	}
	return err
}

func Set(name, value string) error {
	dat, err := json.Marshal(Config)
	if err == nil {
		m := map[string]interface{}{}
		err = json.Unmarshal(dat, &m)
		if err == nil {
			if name == "tlsnoverify" {
				// hack
				m[name] = value == "true"
			} else {
				m[name] = value
			}
			dat, err = json.Marshal(m)
			if err == nil {
				err = json.Unmarshal(dat, Config)
			}
		}
	}
	return err
}

func Load() {
	if ConfigDir == "" {
		ConfigDir = os.Getenv("SOLUBLE_CONFIG_DIR")
		if ConfigDir == "" {
			ConfigDir, _ = homedir.Expand("~/.config/lacework")
		}
	}
	if ConfigFile == "" {
		ConfigFile = os.Getenv("SOLUBLE_CONFIG_FILE")
		if ConfigFile == "" {
			ConfigFile = filepath.Join(ConfigDir, "cli-config.json")
			if !util.FileExists(ConfigFile) {
				legacyConfigFile, _ := homedir.Expand("~/.soluble/cli-config.json")
				if util.FileExists(legacyConfigFile) {
					configFileRead = legacyConfigFile
					migrationAvailable = true
					log.Warnf("Using legacy config file {warning:%s}, use {primary:soluble config migrate} to migrate", configFileRead)
				}
			}
		}
	}
	if info, err := os.Stat(ConfigFile); err == nil && (info.Mode()&0077) != 0 {
		if runtime.GOOS != "windows" {
			if err := os.Chmod(ConfigFile, 0600); err == nil {
				log.Infof("Removed world & group permissions on config file {info:%s}", ConfigFile)
			} else {
				log.Warnf("Config file {info:%s} is world readable could not change permissions: {warning:%s}",
					ConfigFile, err.Error())
			}
		}
	}
	if configFileRead == "" {
		configFileRead = ConfigFile
	}
	dat, err := os.ReadFile(configFileRead)
	if err != nil {
		configFileRead = ""
	} else {
		_ = json.Unmarshal(dat, GlobalConfig)
	}
	if GlobalConfig.Profiles == nil {
		GlobalConfig.Profiles = map[string]*ProfileT{}
	}
	Config = GlobalConfig.Profiles[GlobalConfig.CurrentProfile]
	if Config == nil {
		SelectProfile("default")
	}
	if Config.ProfileName == "" {
		Config.ProfileName = GlobalConfig.CurrentProfile
	}
}

func UpdateFromServerProfile(result *jnode.Node) bool {
	changed := setIfChanged(&Config.Organization, result.Path("currentOrgId").AsText())
	changed = setIfChanged(&Config.Email, result.Path("email").AsText()) || changed
	return changed
}

func setIfChanged(c *string, v string) bool {
	if *c != v {
		*c = v
		return true
	}
	return false
}

func GetModelLocations() []string {
	return GlobalConfig.ModelLocations
}

func Migrate() error {
	if !migrationAvailable {
		log.Infof("Config file {info:%s} is already up-to-date", ConfigFile)
		return nil
	}
	return Save()
}
