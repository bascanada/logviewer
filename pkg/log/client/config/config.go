package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/ty"
)

// ErrContextNotFound is a sentinel error allowing callers to detect missing contexts via errors.Is.
var ErrContextNotFound = errors.New("context not found")

// Sentinel errors returned by LoadContextConfig so callers can detect exact
// failure modes using errors.Is().
var (
	ErrConfigParse = errors.New("invalid config content")
	ErrNoContexts  = errors.New("no contexts found in config file")
	ErrNoClients   = errors.New("no clients found in config file")
)

const (
	// EnvConfigPath is the environment variable used to override the config path
	EnvConfigPath = "LOGVIEWER_CONFIG"

	// DefaultConfigDir is the directory under the user's home where the config
	// file is expected when no explicit path or env var is provided.
	DefaultConfigDir = ".logviewer"

	// DefaultConfigFile is the config filename to look for in the default dir.
	DefaultConfigFile = "config.yaml"
)

func LoadContextConfig(configPath string) (*ContextConfig, error) {
	// If no path provided, first check LOGVIEWER_CONFIG env var, then default $HOME/.logviewer/config.yaml
	if strings.TrimSpace(configPath) == "" {
		if envPath := strings.TrimSpace(os.Getenv(EnvConfigPath)); envPath != "" {
			configPath = envPath
		} else if home, err := os.UserHomeDir(); err == nil {
			defaultPath := filepath.Join(home, DefaultConfigDir, DefaultConfigFile)
			if _, err := os.Stat(defaultPath); err == nil {
				configPath = defaultPath
			}
		}
	}

	if strings.TrimSpace(configPath) == "" {
		return nil, fmt.Errorf("config file not found at path: %s", configPath)
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found at path: %s", configPath)
	}

	// Read file contents and support JSON or YAML formats
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var config ContextConfig
	ext := strings.ToLower(filepath.Ext(configPath))
	switch ext {
	case ".json":
		if err := json.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("%w: parsing JSON %s: %v", ErrConfigParse, configPath, err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, fmt.Errorf("%w: parsing YAML %s: %v", ErrConfigParse, configPath, err)
		}
	default:
		// Try JSON then YAML as a fallback
		if err := json.Unmarshal(data, &config); err == nil {
			break
		}
		if err := yaml.Unmarshal(data, &config); err == nil {
			break
		}
		return nil, fmt.Errorf("%w: unsupported or invalid config format for file: %s", ErrConfigParse, configPath)
	}

	if len(config.Contexts) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrNoContexts, configPath)
	}

	// Ensure the clients map exists and provide a default "local" client
	if config.Clients == nil {
		config.Clients = Clients{}
	}
	if _, ok := config.Clients["local"]; !ok {
		config.Clients["local"] = Client{Type: "local", Options: ty.MI{}}
	}

	if err := validateClients(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// validateClients performs lightweight validation of configured clients and
// returns a combined error describing any missing required options. This is
// intended to help users detect common config typos (e.g. using "option"
// instead of "options") and missing fields such as Url/Endpoint/Addr.
func validateClients(cc *ContextConfig) error {
	problems := []string{}

	for name, c := range cc.Clients {
		switch strings.ToLower(c.Type) {
		case "splunk":
			if c.Options.GetString("url") == "" {
				problems = append(problems, fmt.Sprintf("client '%s' (splunk) missing required option 'url'", name))
			}
		case "opensearch", "kibana":
			if c.Options.GetString("endpoint") == "" {
				problems = append(problems, fmt.Sprintf("client '%s' (%s) missing required option 'endpoint'", name, c.Type))
			}
		case "ssh":
			if c.Options.GetString("addr") == "" {
				problems = append(problems, fmt.Sprintf("client '%s' (ssh) missing required option 'addr'", name))
			}
		case "docker":
			// docker Host can be empty (falls back to unix socket), so just warn
			// but do not fail.
			// no-op
		default:
			// Unknown types are not validated here.
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid client configuration:\n  %s", strings.Join(problems, "\n  "))
	}
	return nil
}

type Client struct {
	Type    string `json:"type"`
	Options ty.MI  `json:"options"`
}

type SearchContext struct {
	Client        string           `json:"client" yaml:"client"`
	SearchInherit []string         `json:"searchInherit" yaml:"searchInherit"`
	Search        client.LogSearch `json:"search" yaml:"search"`
}

type Clients map[string]Client

type Searches map[string]client.LogSearch

type Contexts map[string]SearchContext

type ContextConfig struct {
	Clients  `json:"clients" yaml:"clients"`
	Searches `json:"searches" yaml:"searches"`
	Contexts `json:"contexts" yaml:"contexts"`
}

func (cc ContextConfig) GetSearchContext(contextId string, inherits []string, logSearch client.LogSearch, runtimeVars map[string]string) (SearchContext, error) {
	if contextId == "" {
		return SearchContext{}, errors.New("contextId is empty, required when using config")
	}

	searchContext, ok := cc.Contexts[contextId]
	if !ok {
		return SearchContext{}, fmt.Errorf("%w: %s", ErrContextNotFound, contextId)
	}

	// Combine inherits from context and the call
	allInherits := append(searchContext.SearchInherit, inherits...)
	if len(allInherits) > 0 {
		for _, inherit := range allInherits {
			inheritSearch, found := cc.Searches[inherit]
			if !found {
				return SearchContext{}, fmt.Errorf("failed to find a search context for %s", inherit)
			}
			searchContext.Search.MergeInto(&inheritSearch)
		}
	}

	// Merge the provided logSearch into the context's search
	searchContext.Search.MergeInto(&logSearch)

	// Build complete variable map: defaults from variable definitions + runtime vars (runtime takes precedence)
	completeVars := make(map[string]string)
	// First, add defaults from variable definitions
	for varName, varDef := range searchContext.Search.Variables {
		if varDef.Default != nil {
			completeVars[varName] = fmt.Sprintf("%v", varDef.Default)
		}
	}
	// Then, override with runtime variables
	for k, v := range runtimeVars {
		completeVars[k] = v
	}

	// Resolve variables in all relevant fields
	searchContext.Search.Fields = searchContext.Search.Fields.ResolveVariablesWith(completeVars)
	searchContext.Search.FieldsCondition = searchContext.Search.FieldsCondition.ResolveVariablesWith(completeVars)
	searchContext.Search.Options = searchContext.Search.Options.ResolveVariablesWith(completeVars)

	// Resolve variables in Opt[string] fields
	if searchContext.Search.PrinterOptions.Template.Set {
		resolvedTemplate := ty.ResolveVars(searchContext.Search.PrinterOptions.Template.Value, completeVars)
		searchContext.Search.PrinterOptions.Template.S(resolvedTemplate)
	}
	if searchContext.Search.PrinterOptions.MessageRegex.Set {
		resolvedMessageRegex := ty.ResolveVars(searchContext.Search.PrinterOptions.MessageRegex.Value, completeVars)
		searchContext.Search.PrinterOptions.MessageRegex.S(resolvedMessageRegex)
	}
	if searchContext.Search.FieldExtraction.GroupRegex.Set {
		resolvedGroupRegex := ty.ResolveVars(searchContext.Search.FieldExtraction.GroupRegex.Value, completeVars)
		searchContext.Search.FieldExtraction.GroupRegex.S(resolvedGroupRegex)
	}
	if searchContext.Search.FieldExtraction.KvRegex.Set {
		resolvedKvRegex := ty.ResolveVars(searchContext.Search.FieldExtraction.KvRegex.Value, completeVars)
		searchContext.Search.FieldExtraction.KvRegex.S(resolvedKvRegex)
	}
	if searchContext.Search.FieldExtraction.TimestampRegex.Set {
		resolvedTimestampRegex := ty.ResolveVars(searchContext.Search.FieldExtraction.TimestampRegex.Value, completeVars)
		searchContext.Search.FieldExtraction.TimestampRegex.S(resolvedTimestampRegex)
	}

	return searchContext, nil
}
