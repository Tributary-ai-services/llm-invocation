package types

type Config struct {
	Providers map[string]ProviderConfig `json:"providers"`
	Tools     ToolsConfig               `json:"tools"`
	MCP       MCPConfig                 `json:"mcp"`
	Defaults  DefaultsConfig            `json:"defaults"`
}

type ProviderConfig struct {
	Enabled bool                   `json:"enabled"`
	APIKey  string                 `json:"api_key"`
	BaseURL string                 `json:"base_url,omitempty"`
	Models  []string               `json:"models,omitempty"`
	Options map[string]interface{} `json:"options,omitempty"`
	Timeout int                    `json:"timeout,omitempty"`
}

type ToolsConfig struct {
	Enabled   bool     `json:"enabled"`
	Whitelist []string `json:"whitelist,omitempty"`
	Blacklist []string `json:"blacklist,omitempty"`
	Sandbox   bool     `json:"sandbox"`
}

type MCPConfig struct {
	Enabled bool              `json:"enabled"`
	Servers []MCPServerConfig `json:"servers,omitempty"`
}

type MCPServerConfig struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

type DefaultsConfig struct {
	Provider string       `json:"provider,omitempty"`
	Model    string       `json:"model,omitempty"`
	Options  ModelOptions `json:"options,omitempty"`
}

func DefaultConfig() *Config {
	return &Config{
		Providers: make(map[string]ProviderConfig),
		Tools: ToolsConfig{
			Enabled: true,
			Sandbox: true,
		},
		MCP: MCPConfig{
			Enabled: false,
		},
		Defaults: DefaultsConfig{},
	}
}