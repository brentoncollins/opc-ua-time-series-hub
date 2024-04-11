package util

import (
	"os"
)

type Config struct {
	LogFilePath                  string
	DbUser                       string
	DbPassword                   string
	DbHost                       string
	DbPort                       string
	DbName                       string
	TelegrafConfigPath           string
	TelegrafInterval             string
	TelegrafRoundInterval        string
	TelegrafMetricBatchSize      string
	TelegrafMetricBufferLimit    string
	TelegrafInfluxUrl            string
	TelegrafInfluxToken          string
	TelegrafInfluxOrg            string
	TelegrafInfluxBucket         string
	TelegrafOpcUaEndpoint        string
	RootNode                     string
	TelegrafCollectionJitter     string
	TelegrafFlushInterval        string
	TelegrafFlushJitter          string
	TelegrafPrecision            string
	TelegrafInputsTotalCPU       string
	TelegrafInputsCollectCpuTime string
	TelegrafInputsReportActive   string
	TelegrafInputsCoreTags       string
	TelegrafInputsPerCPU         string
}

func LoadConfig() Config {
	return Config{
		LogFilePath:                  getEnv("LOG_FILE_PATH", ""),
		DbUser:                       getEnv("DB_USER", ""),
		DbPassword:                   getEnv("DB_PASSWORD", ""),
		DbHost:                       getEnv("DB_HOST", ""),
		DbPort:                       getEnv("DB_PORT", ""),
		DbName:                       getEnv("DB_NAME", ""),
		TelegrafConfigPath:           getEnv("TELEGRAF_CONFIG_FILE", ""),
		TelegrafInterval:             getEnv("TELEGRAF_INTERVAL", ""),
		TelegrafRoundInterval:        getEnv("TELEGRAF_ROUND_INTERVAL", ""),
		TelegrafMetricBatchSize:      getEnv("TELEGRAF_METRIC_BATCH_SIZE", ""),
		TelegrafMetricBufferLimit:    getEnv("TELEGRAF_METRIC_BUFFER_LIMIT", ""),
		TelegrafCollectionJitter:     getEnv("TELEGRAF_COLLECTION_JITTER", ""),
		TelegrafFlushInterval:        getEnv("TELEGRAF_FLUSH_INTERVAL", ""),
		TelegrafFlushJitter:          getEnv("TELEGRAF_FLUSH_JITTER", ""),
		TelegrafPrecision:            getEnv("TELEGRAF_PRECISION", ""),
		TelegrafInfluxUrl:            getEnv("TELEGRAF_INFLUX_URL", ""),
		TelegrafInfluxToken:          getEnv("TELEGRAF_INFLUX_TOKEN", ""),
		TelegrafInfluxOrg:            getEnv("TELEGRAF_INFLUX_ORG", ""),
		TelegrafInfluxBucket:         getEnv("TELEGRAF_INFLUX_BUCKET", ""),
		TelegrafOpcUaEndpoint:        getEnv("TELEGRAF_OPCUA_ENDPOINT", ""),
		TelegrafInputsPerCPU:         getEnv("TELEGRAF_PER_CPU", ""),
		TelegrafInputsTotalCPU:       getEnv("TELEGRAF_TOTAL_CPU", ""),
		TelegrafInputsCollectCpuTime: getEnv("TELEGRAF_COLLECT_CPU_TIME", ""),
		TelegrafInputsReportActive:   getEnv("TELEGRAF_REPORT_ACTIVE", ""),
		TelegrafInputsCoreTags:       getEnv("TELEGRAF_CORE_TAGS", ""),
		RootNode:                     getEnv("ROOT_NODE", ""),
	}
}

func getEnv(key string, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		Logger.Infof("Environment variable %s set to %s", key, value)
		return value
	}
	if defaultValue == "" {
		Logger.Fatalf("Environment variable %s not set", key)
		os.Exit(1)
	}
	return defaultValue
}
