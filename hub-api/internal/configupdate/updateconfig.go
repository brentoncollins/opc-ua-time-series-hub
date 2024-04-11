package configupdate

import (
	"OpcUaTimeSeriesHub/hub-api/internal/database"
	"bytes"
	"github.com/BurntSushi/toml"
	"log"
	"os"
	"strconv"
)

type Config struct {
	GlobalTags map[string]string   `toml:"global_tags"`
	Agent      Agent               `toml:"agent"`
	Outputs    map[string][]Output `toml:"outputs"`
	Inputs     Inputs              `toml:"inputs"`
}

type Agent struct {
	Interval          string `toml:"interval"`
	RoundInterval     bool   `toml:"round_interval"`
	MetricBatchSize   int    `toml:"metric_batch_size"`
	MetricBufferLimit int    `toml:"metric_buffer_limit"`
	CollectionJitter  string `toml:"collection_jitter"`
	FlushInterval     string `toml:"flush_interval"`
	FlushJitter       string `toml:"flush_jitter"`
	Precision         string `toml:"precision"`
	Debug             bool   `toml:"debug"`
}

type Output struct {
	URLs         []string `toml:"urls"`
	Token        string   `toml:"token"`
	Organization string   `toml:"organization"`
	Bucket       string   `toml:"bucket"`
}

type CPU struct {
	PerCPU         bool `toml:"percpu"`
	TotalCPU       bool `toml:"totalcpu"`
	CollectCPUTime bool `toml:"collect_cpu_time"`
	ReportActive   bool `toml:"report_active"`
	CoreTags       bool `toml:"core_tags"`
}

type Inputs struct {
	CPU   []CPU   `toml:"cpu"`
	Opcua []Opcua `toml:"opcua"`
}

type Opcua struct {
	Endpoint       string        `toml:"endpoint"`
	ConnectTimeout string        `toml:"connect_timeout"`
	RequestTimeout string        `toml:"request_timeout"`
	SecurityPolicy string        `toml:"security_policy"`
	SecurityMode   string        `toml:"security_mode"`
	Nodes          []*SimpleNode `toml:"nodes"`
}

type SimpleNode struct {
	Name           string `toml:"name"`
	Namespace      string `toml:"namespace"`
	IdentifierType string `toml:"identifier_type"`
	Identifier     string `toml:"identifier"`
}

func ConvertToSimpleNodes(nodes []*database.Node) []*SimpleNode {
	simpleNodes := make([]*SimpleNode, len(nodes))
	for i, node := range nodes {
		simpleNodes[i] = &SimpleNode{
			Name:           node.BrowseName,
			Namespace:      strconv.Itoa(node.Namespace),
			IdentifierType: node.IdentifierType,
			Identifier:     node.Identifier,
		}
	}
	return simpleNodes
}

func UpdateConfig(configFile string, nodes []*database.Node) error {

	simpleNodes := ConvertToSimpleNodes(nodes)

	// Load TOML file
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
		return err
	}

	// Unmarshal the TOML data into our Config struct
	var config Config
	if _, err := toml.Decode(string(data), &config); err != nil {
		log.Fatal(err)
		return err
	}

	// If there are nodes, replace the inputs.opcua section with new data
	if len(simpleNodes) > 0 {
		config.Inputs.Opcua = []Opcua{
			{
				Endpoint:       os.Getenv("TELEGRAF_OPCUA_ENDPOINT"),
				ConnectTimeout: "10s",
				RequestTimeout: "5s",
				SecurityPolicy: "None",
				SecurityMode:   "None",
				Nodes:          simpleNodes,
			},
		}
	} else {
		// If there are no nodes, remove the inputs.opcua section entirely
		config.Inputs.Opcua = nil
	}

	// Marshal the modified config back to TOML
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(config); err != nil {
		log.Fatal(err)
		return err
	}

	// Write the new TOML to the file (or to a new file)
	if err := os.WriteFile(configFile, buf.Bytes(), 0644); err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func CreateConfig(configFile string) error {
	cfg := Config{
		GlobalTags: map[string]string{},
		Agent: Agent{
			Interval:          os.Getenv("TELEGRAF_INTERVAL"),
			RoundInterval:     os.Getenv("TELEGRAF_ROUND_INTERVAL") == "true",
			MetricBatchSize:   atoi(os.Getenv("TELEGRAF_METRIC_BATCH_SIZE")),
			MetricBufferLimit: atoi(os.Getenv("TELEGRAF_METRIC_BUFFER_LIMIT")),
			CollectionJitter:  os.Getenv("TELEGRAF_COLLECTION_JITTER"),
			FlushInterval:     os.Getenv("TELEGRAF_FLUSH_INTERVAL"),
			FlushJitter:       os.Getenv("TELEGRAF_FLUSH_JITTER"),
			Precision:         os.Getenv("TELEGRAF_PRECISION"),
			Debug:             false,
		},
		Outputs: map[string][]Output{
			"influxdb_v2": {
				{
					URLs:         []string{os.Getenv("TELEGRAF_INFLUX_URL")},
					Token:        os.Getenv("TELEGRAF_INFLUX_TOKEN"),
					Organization: os.Getenv("TELEGRAF_INFLUX_ORG"),
					Bucket:       os.Getenv("TELEGRAF_INFLUX_BUCKET"),
				},
			},
		},
		Inputs: Inputs{
			CPU: []CPU{
				{
					PerCPU:         os.Getenv("TELEGRAF_CPU_PERCPU") == "true",
					TotalCPU:       os.Getenv("TELEGRAF_TOTAL_CPU") == "true",
					CollectCPUTime: os.Getenv("TELEGRAF_COLLECT_CPU_TIME") == "true",
					ReportActive:   os.Getenv("TELEGRAF_REPORT_ACTIVE") == "true",
					CoreTags:       os.Getenv("TELEGRAF_CORE_TAGS") == "true",
				},
			},
		},
	}

	cfg.Inputs.Opcua = nil
	// Marshal the modified config back to TOML
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		log.Fatal(err)
		return err
	}

	// Write the new TOML to the file (or to a new file)
	if err := os.WriteFile(configFile, buf.Bytes(), 0644); err != nil {
		log.Fatal(err)
		return err
	}
	return nil
}

func atoi(s string) int {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	return 0
}
