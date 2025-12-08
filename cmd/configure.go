// SPDX-License-Identifier: GPL-3.0-only
package cmd

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bascanada/logviewer/pkg/log/client"
	"github.com/bascanada/logviewer/pkg/log/client/config"
	"github.com/bascanada/logviewer/pkg/ty"
	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Interactive wizard to generate a configuration file",
	Long: `Launch an interactive wizard to help you create your first logviewer configuration.

This command will guide you through setting up a log source (Splunk, OpenSearch, 
Kubernetes, Docker, SSH, or CloudWatch) and generate a ready-to-use config file.

Example:
  logviewer configure
  logviewer configure -c /path/to/config.yaml`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runConfigWizard(configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)
}

func runConfigWizard(cfgPath string) error {
	var (
		clientType string
		clientName string
		endpoint   string
		authType   string
		token      string
		username   string
		password   string
		sshAddr    string
		sshUser    string
		sshKey     string
		region     string
		kubeConfig string
		confirm    bool
	)

	// Welcome message
	fmt.Println("🚀 Welcome to logviewer configuration wizard!")
	fmt.Println()

	// 1. Basic Information Form
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Which log source do you want to configure?").
				Description("Select the type of log backend you'll be querying").
				Options(
					huh.NewOption("Splunk", "splunk"),
					huh.NewOption("OpenSearch / Elasticsearch", "opensearch"),
					huh.NewOption("Kubernetes", "k8s"),
					huh.NewOption("Docker", "docker"),
					huh.NewOption("SSH (Remote Command)", "ssh"),
					huh.NewOption("AWS CloudWatch", "cloudwatch"),
					huh.NewOption("Local Command", "local"),
				).
				Value(&clientType),

			huh.NewInput().
				Title("Name for this client").
				Description("A friendly name to identify this log source (e.g., production-splunk, local-docker)").
				Placeholder("my-log-source").
				Value(&clientName).
				Validate(func(str string) error {
					if strings.TrimSpace(str) == "" {
						return fmt.Errorf("name cannot be empty")
					}
					if strings.ContainsAny(str, " \t\n") {
						return fmt.Errorf("name cannot contain whitespace")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	// 2. Dynamic fields based on selection
	switch clientType {
	case "splunk":
		if err := configureSplunk(&endpoint, &authType, &token, &username, &password); err != nil {
			return err
		}
	case "opensearch":
		if err := configureOpenSearch(&endpoint, &username, &password); err != nil {
			return err
		}
	case "ssh":
		if err := configureSSH(&sshAddr, &sshUser, &sshKey); err != nil {
			return err
		}
	case "cloudwatch":
		if err := configureCloudWatch(&region, &endpoint); err != nil {
			return err
		}
	case "k8s":
		if err := configureKubernetes(&kubeConfig); err != nil {
			return err
		}
	case "docker":
		// Docker typically uses default socket, no additional config needed
		fmt.Println("✓ Docker will use the default Unix socket (unix:///var/run/docker.sock)")
	case "local":
		// Local file reader, no additional config needed
		fmt.Println("✓ Local file client configured (will read files directly)")
	}

	// 3. Construct the Config Object
	cfg := config.ContextConfig{
		Clients:  make(config.Clients),
		Contexts: make(config.Contexts),
		Searches: make(config.Searches),
	}

	// Build Client Options
	opts := buildClientOptions(clientType, endpoint, authType, token, username, password,
		sshAddr, sshUser, sshKey, region, kubeConfig)

	// Add Client
	cfg.Clients[clientName] = config.Client{
		Type:    clientType,
		Options: opts,
	}

	// Add a Default Context based on client type
	contextName := clientName + "-default"
	searchConfig := buildDefaultSearch(clientType)
	cfg.Contexts[contextName] = config.SearchContext{
		Client: clientName,
		Search: searchConfig,
	}

	// 4. Preview Configuration
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to generate YAML: %w", err)
	}

	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Println("📝 Generated Configuration:")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println(string(out))
	fmt.Println(strings.Repeat("─", 60) + "\n")

	// 5. Confirm and Save
	// Determine target config path for confirmation message
	var targetPath string
	if strings.TrimSpace(cfgPath) != "" {
		targetPath = cfgPath
	} else if envPath := strings.TrimSpace(os.Getenv(config.EnvConfigPath)); envPath != "" {
		targetPath = envPath
	} else {
		home, _ := os.UserHomeDir()
		targetPath = filepath.Join(home, config.DefaultConfigDir, config.DefaultConfigFile)
	}

	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Save this configuration?").
				Description(fmt.Sprintf("Target: %s", targetPath)).
				Affirmative("Yes, save it!").
				Negative("No, cancel").
				Value(&confirm),
		),
	)

	if err := confirmForm.Run(); err != nil {
		return err
	}

	if !confirm {
		fmt.Println("❌ Configuration not saved. Run 'logviewer configure' again when ready.")
		return nil
	}

	// Determine config path from flag, env var, or default
	var configPath string
	if strings.TrimSpace(cfgPath) != "" {
		configPath = cfgPath
	} else if envPath := strings.TrimSpace(os.Getenv(config.EnvConfigPath)); envPath != "" {
		configPath = envPath
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(home, config.DefaultConfigDir, config.DefaultConfigFile)
	}

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Check if config already exists
	isNewFile := true
	if _, err := os.Stat(configPath); err == nil {
		isNewFile = false
	}

	// Display file status before saving
	if isNewFile {
		fmt.Printf("\n📄 Creating new configuration file: %s\n", configPath)
	} else {
		fmt.Printf("\n📝 Updating existing configuration file: %s\n", configPath)
	}

	// Check if config already exists and merge if needed
	existingCfg, err := config.LoadContextConfig(configPath)
	if err == nil && existingCfg != nil {
		// Merge with existing config
		for k, v := range cfg.Clients {
			existingCfg.Clients[k] = v
		}
		for k, v := range cfg.Contexts {
			existingCfg.Contexts[k] = v
		}
		out, err = yaml.Marshal(existingCfg)
		if err != nil {
			return fmt.Errorf("failed to merge with existing config: %w", err)
		}
	}

	if err := os.WriteFile(configPath, out, 0644); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Success message with next steps
	fmt.Printf("\n✅ Configuration saved to %s\n\n", configPath)
	fmt.Println("🎉 You're all set! Try it now:")
	fmt.Printf("   logviewer query -i %s\n\n", contextName)

	if clientType == "local" {
		fmt.Println("💡 For local files, you'll need to specify a command in your context.")
		fmt.Println("   Edit your config and add an 'options.cmd' field, for example:")
		fmt.Println("   options:")
		fmt.Println("     cmd: 'tail -n 100 /path/to/your/logfile.log'")
	}

	return nil
}

func configureSplunk(endpoint, authType, token, username, password *string) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Splunk URL").
				Description("Full URL to your Splunk API endpoint").
				Placeholder("https://splunk.example.com:8089/services").
				Value(endpoint).
				Validate(func(str string) error {
					if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
						return fmt.Errorf("URL must start with http:// or https://")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("Authentication method").
				Options(
					huh.NewOption("Splunk Token", "splunk"),
					huh.NewOption("Bearer Token (Username/Password MD5)", "bearer"),
					huh.NewOption("Bearer Token (Pre-computed Hash)", "bearer-hash"),
				).
				Value(authType),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	// Auth details
	switch *authType {
	case "splunk":
		authForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Splunk Token").
					Description("Your Splunk authentication token").
					Value(token).
					EchoMode(huh.EchoModePassword),
			),
		)
		return authForm.Run()
	case "bearer-hash":
		authForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Bearer Token Hash").
					Description("Your pre-computed MD5 hash").
					Value(token).
					EchoMode(huh.EchoModePassword),
			),
		)
		return authForm.Run()
	default:
		authForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Username").
					Value(username),
				huh.NewInput().
					Title("Password").
					Value(password).
					EchoMode(huh.EchoModePassword),
			),
		)
		return authForm.Run()
	}
}

func configureOpenSearch(endpoint, username, password *string) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("OpenSearch Endpoint").
				Description("URL to your OpenSearch/Elasticsearch cluster").
				Placeholder("http://localhost:9200").
				Value(endpoint).
				Validate(func(str string) error {
					if !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
						return fmt.Errorf("URL must start with http:// or https://")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	// Optional auth
	var needsAuth bool
	authQuestion := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Does your OpenSearch require authentication?").
				Value(&needsAuth),
		),
	)

	if err := authQuestion.Run(); err != nil {
		return err
	}

	if needsAuth {
		authForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Username").
					Value(username),
				huh.NewInput().
					Title("Password").
					Value(password).
					EchoMode(huh.EchoModePassword),
			),
		)
		return authForm.Run()
	}

	return nil
}

func configureSSH(addr, user, key *string) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("SSH Address").
				Description("Host and port for SSH connection").
				Placeholder("hostname:22").
				Value(addr),
			huh.NewInput().
				Title("SSH Username").
				Value(user),
			huh.NewInput().
				Title("Private Key Path").
				Description("Path to your SSH private key file (optional, will use default if empty)").
				Placeholder("~/.ssh/id_rsa").
				Value(key),
		),
	).Run()
}

func configureCloudWatch(region, endpoint *string) error {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("AWS Region").
				Description("AWS region for CloudWatch logs").
				Placeholder("us-east-1").
				Value(region),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	// Optional custom endpoint (for LocalStack)
	var useCustomEndpoint bool
	customQuestion := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Use custom endpoint? (e.g., for LocalStack)").
				Value(&useCustomEndpoint),
		),
	)

	if err := customQuestion.Run(); err != nil {
		return err
	}

	if useCustomEndpoint {
		return huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Custom Endpoint").
					Placeholder("http://localhost:4566").
					Value(endpoint),
			),
		).Run()
	}

	return nil
}

func configureKubernetes(kubeConfig *string) error {
	var useCustomConfig bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Use custom kubeconfig path?").
				Description("Default uses ~/.kube/config").
				Value(&useCustomConfig),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if useCustomConfig {
		return huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Kubeconfig Path").
					Placeholder("~/.kube/config").
					Value(kubeConfig),
			),
		).Run()
	}

	return nil
}

func buildClientOptions(clientType, endpoint, authType, token, username, password,
	sshAddr, sshUser, sshKey, region, kubeConfig string) ty.MI {

	opts := ty.MI{}

	switch clientType {
	case "splunk":
		opts["url"] = endpoint
		headers := ty.MS{}

		if authType == "splunk" {
			headers["Authorization"] = "Splunk " + token
		} else if authType == "bearer-hash" {
			// Use pre-computed hash directly
			headers["Authorization"] = "Bearer " + token
		} else {
			// Calculate MD5 hash of username:password for Bearer auth
			hash := md5.Sum([]byte(username + ":" + password))
			hashStr := hex.EncodeToString(hash[:])
			headers["Authorization"] = "Bearer " + hashStr
		}

		opts["headers"] = headers
		opts["searchBody"] = ty.MI{"output_mode": "json"}
		opts["usePollingFollow"] = false

	case "opensearch":
		opts["endpoint"] = endpoint
		if username != "" && password != "" {
			opts["auth"] = ty.MI{
				"username": username,
				"password": password,
			}
		}

	case "ssh":
		opts["addr"] = sshAddr
		opts["user"] = sshUser
		if sshKey != "" {
			opts["privateKey"] = sshKey
		}

	case "cloudwatch":
		opts["region"] = region
		if endpoint != "" {
			opts["endpoint"] = endpoint
		}

	case "k8s":
		if kubeConfig != "" {
			opts["kubeConfig"] = kubeConfig
		}

	case "docker":
		opts["host"] = "unix:///var/run/docker.sock"

	case "local":
		// Local client doesn't need options at the client level
		// Commands are specified in the context
	}

	return opts
}

func buildDefaultSearch(clientType string) client.LogSearch {
	search := client.LogSearch{
		Size:   ty.OptWrap(100),
		Fields: ty.MS{},
	}

	switch clientType {
	case "splunk":
		search.Options = ty.MI{
			"index": "main",
		}

	case "opensearch":
		search.Options = ty.MI{
			"index": "logs-*",
		}

	case "ssh", "local":
		// For SSH and local, the command needs to be specified
		// We'll provide a placeholder
		search.Options = ty.MI{
			"cmd": "tail -n {{or .Size.Value 100}} /path/to/logfile.log",
		}

	case "cloudwatch":
		search.Options = ty.MI{
			"logGroup": "/aws/lambda/my-function",
		}

	case "docker":
		search.Options = ty.MI{
			"container": "container-name-or-id",
		}

	case "k8s":
		search.Options = ty.MI{
			"namespace": "default",
			"pod":       "pod-name",
		}
	}

	return search
}
