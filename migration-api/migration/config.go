package migration

import (
	"encoding/json"
	"time"

	"github.com/spf13/viper"
)

// ReadConfig reads the config.
func ReadConfig() (*Config, error) {
	viper.SetConfigFile("config.json")
	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	c := &Config{}
	if err := viper.Unmarshal(c); err != nil {
		return nil, err
	}
	return c, nil
}

type Config struct {
	SharedListenAddress string                   `json:"sharedListenAddress"`
	ShutdownMaxAwait    time.Duration            `json:"shutdownMaxAwait"`
	HTTPAPIService      HTTPAPIServiceConfig     `json:"httpAPIService"`
	PromMetricsService  PromMetricsServiceConfig `json:"promMetricsService"`
}

func (c *Config) JSONString() string {
	configJson, err := json.MarshalIndent(c, "", "   ")
	if err != nil {
		panic(err)
	}
	return string(configJson)
}

type HTTPAPIServiceConfig struct {
	MaxMilestonesToQueryForEntries int              `json:"maxMilestonesToQueryForEntries"`
	MinTokenAmountForMigration     int              `json:"minTokenAmountForMigration"`
	LegacyNode                     LegacyNodeConfig `json:"legacyNode"`
	C2Node                         C2NodeConfig     `json:"c2Node"`
}

type LegacyNodeConfig struct {
	URI     string        `json:"uri"`
	Timeout time.Duration `json:"timeout"`
}

type C2NodeConfig struct {
	URI     string        `json:"uri"`
	Timeout time.Duration `json:"timeout"`
}

type PromMetricsServiceConfig struct {
	Enabled                   bool   `json:"enabled"`
	Debug                     bool   `json:"debug"`
	LegacyMilestoneStartIndex int    `json:"legacyMilestoneStartIndex"`
	C2MilestoneStartIndex     int    `json:"c2MilestoneStartIndex"`
	StateFilePath             string `json:"stateFilePath"`
	CounterNames              struct {
		ServiceErrors         string `json:"serviceErrors"`
		IncludedLegacyTails   string `json:"includedLegacyTails"`
		AppliedReceiptEntries string `json:"appliedReceiptEntries"`
	} `json:"counterNames"`
	FetchInterval time.Duration    `json:"fetchInterval"`
	LegacyNode    LegacyNodeConfig `json:"legacyNode"`
	C2Node        C2NodeConfig     `json:"c2Node"`
}
