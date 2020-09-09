package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

const (
	defConfigPath = "/tmp"
)

type ClusterRootConfig struct {
	Version string         `yaml:"version,omitempty"`
	Cluster *ClusterConfig `yaml:"cluster"`
}

type ClusterConfig struct {
	Name        string             `yaml:"name"`
	ConsulAddr  string             `yaml:"consulAddr,omitempty"`
	LogLevel    string             `yaml:"logLevel,omitempty"`
	LogDir      string             `yaml:"logDir,omitempty"`
	WarnLogDir  string             `yaml:"warnLogDir,omitempty"`
	AppHome     string             `yaml:"appHome,omitempty"`
	Passwd      string             `yaml:"passwd,omitempty"`
	Master      *MasterConfig      `yaml:"master"`
	MetaNode    *MetaNodeConfig    `yaml:"metanode"`
	DataNode    *DataNodeConfig    `yaml:"datanode"`
	ObjectNode  *ObjectNodeConfig  `yaml:"objectnode,omitempty"`
	ConsoleNode *ConsoleNodeConfig `yaml:"consolenode,omitempty"`
	Client      *ClientConfig      `yaml:"client,omitempty"`
}

type MasterConfig struct {
	Hosts        []string `yaml:"hosts" json:"-"`
	Passwd       string   `yaml:"passwd,omitempty" json:"-"`
	Listen       string   `yaml:"listen" json:"listen"`
	Prof         string   `yaml:"prof" json:"prof"`
	RetainLogs   string   `yaml:"retainLogs" json:"retainLogs"`
	LogDir       string   `yaml:"logDir" json:"logDir"`
	LogLevel     string   `yaml:"logLevel,omitempty" json:"logLevel"`
	WarnLogDir   string   `yaml:"warnLogDir" json:"warnLogDir"`
	ConsulAddr   string   `yaml:"consulAddr,omitempty" json:"consulAddr,omitempty"`
	ExporterPort string   `yaml:"exporterPort" json:"exporterPort,omitempty"`
	Id           string   `json:"id"`
	Ip           string   `json:"ip"`
	Peers        string   `json:"peers"`
	ClusterName  string   `json:"clusterName"`
}

type MetaNodeConfig struct {
	Hosts            []string `yaml:"hosts" json:"-"`
	Passwd           string   `yaml:"passwd,omitempty" json:"-"`
	MasterAddr       []string `yaml:"-" json:"masterAddr"`
	Listen           string   `yaml:"listen" json:"listen"`
	Prof             string   `yaml:"prof" json:"prof"`
	LogDir           string   `yaml:"logDir" json:"logDir"`
	LogLevel         string   `yaml:"logLevel" json:"logLevel"`
	WarnLogDir       string   `yaml:"warnLogDir" json:"warnLogDir"`
	WalDir           string   `yaml:"walDir" json:"listen"`
	ConsulAddr       string   `yaml:"consulAddr" json:"consulAddr"`
	ExporterPort     string   `yaml:"exporterPort" json:"exporterPort"`
	RaftHeartbetPort string   `yaml:"raftHeartbeatPort" json:"raftHeartbeatPort"`
	RaftReplicaPort  string   `yaml:"raftReplicaPort" json:"raftReplicaPort"`
	StoreType        string   `yaml:"storeType,omitempty" json:"storeType"`
	TotalMem         string   `yaml:"totalMem" json:"totalMem"`
	RocksDirs        []string `yaml:"rocksDirs" json:"rocksDirs"`
	MetadataDir      string   `yaml:"metadataDir" json:"metadataDir"`
	RaftDir          string   `yaml:"raftDir" json:"raftDir"`
}

type DataNodeConfig struct {
	Hosts            []string `yaml:"hosts" json:"-"`
	Passwd           string   `yaml:"passwd,omitempty" json:"-"`
	MasterAddr       []string `yaml:"-" json:"masterAddr"`
	Listen           string   `yaml:"listen" json:"listen"`
	Prof             string   `yaml:"prof" json:"prof"`
	LogDir           string   `yaml:"logDir" json:"logDir"`
	LogLevel         string   `yaml:"logLevel" json:"logLevel"`
	WarnLogDir       string   `yaml:"warnLogDir" json:"warnLogDir"`
	WalDir           string   `yaml:"walDir" json:"listen"`
	ConsulAddr       string   `yaml:"consulAddr" json:"consulAddr"`
	ExporterPort     string   `yaml:"exporterPort" json:"exporterPort"`
	RaftHeartbetPort string   `yaml:"raftHeartbeatPort" json:"raftHeartbeatPort"`
	RaftReplicaPort  string   `yaml:"raftReplicaPort" json:"raftReplicaPort"`
	Disks            []string `yaml:"disks" json:"disks"`
}

type ObjectNodeConfig struct {
	Hosts        []string `yaml:"hosts" json:"-"`
	Passwd       string   `yaml:"passwd,omitempty" json:"-"`
	MasterAddr   []string `yaml:"-" json:"masterAddr"`
	Listen       string   `yaml:"listen" json:"listen"`
	Prof         string   `yaml:"prof" json:"prof"`
	LogDir       string   `yaml:"logDir" json:"logDir"`
	LogLevel     string   `yaml:"logLevel" json:"logLevel"`
	WarnLogDir   string   `yaml:"warnLogDir" json:"warnLogDir"`
	WalDir       string   `yaml:"walDir" json:"listen"`
	ConsulAddr   string   `yaml:"consulAddr" json:"consulAddr"`
	ExporterPort string   `yaml:"exporterPort" json:"exporterPort"`
}

type ConsoleNodeConfig struct {
	Hosts        []string `yaml:"hosts" json:"-"`
	Passwd       string   `yaml:"passwd,omitempty" json:"-"`
	MasterAddr   []string `yaml:"-" json:"masterAddr"`
	Listen       string   `yaml:"listen" json:"listen"`
	Prof         string   `yaml:"prof" json:"prof"`
	LogDir       string   `yaml:"logDir" json:"logDir"`
	LogLevel     string   `yaml:"logLevel" json:"logLevel"`
	WarnLogDir   string   `yaml:"warnLogDir" json:"warnLogDir"`
	ConsulAddr   string   `yaml:"consulAddr" json:"consulAddr"`
	ExporterPort string   `yaml:"exporterPort" json:"exporterPort"`
}

type ClientConfig struct {
	Hosts        []string `yaml:"hosts" json:"-"`
	Passwd       string   `yaml:"passwd,omitempty" json:"-"`
	MasterAddr   string   `yaml:"-" json:"masterAddr"`
	Prof         string   `yaml:"prof" json:"prof"`
	LogDir       string   `yaml:"logDir" json:"logDir"`
	LogLevel     string   `yaml:"logLevel" json:"logLevel"`
	WarnLogDir   string   `yaml:"warnLogDir" json:"warnLogDir"`
	MountPoint   string   `yaml:"mountPoint" json:"mountPoint"`
	volName      string   `yaml:"volName" json:"volName"`
	owner        string   `yaml:"owner" json:"owner"`
	ConsulAddr   string   `yaml:"consulAddr" json:"consulAddr"`
	ExporterPort string   `yaml:"exporterPort" json:"exporterPort"`
}

func ReadYamlConfig(path string) (*ClusterRootConfig, error) {
	conf := &ClusterRootConfig{}
	if f, err := os.Open(path); err != nil {
		stdout(fmt.Sprintf("open %v error: %v\n", path, err))
		return nil, err
	} else {
		if err := yaml.NewDecoder(f).Decode(conf); err != nil {
			stdout(fmt.Sprintf("decode error: %v\n", err))
			return nil, err
		}
	}

	return conf, nil
}

func newDeployCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "deploy",
		Short: "Deploy chubaofs cluster cmd",
	}
	cmd.AddCommand(genClusterConfigCmd())
	//cmd.AddCommand(deployClusterCmd())
	return cmd
}

func upStrIfEmpty(v *string, n string) {
	if *v == "" {
		*v = n
	}
}

func genClusterConfigCmd() *cobra.Command {
	var optClusterConfig string
	//var optTimeout uint16
	var cmd = &cobra.Command{
		Use:   "genConfig",
		Short: "Generate cluster config files",
		Long:  `Set the config file`,
		Run: func(cmd *cobra.Command, args []string) {
			var (
				err        error
				rootConfig *ClusterRootConfig
				//masterHosts []string
			)
			defer func() {
				if err != nil {
					errout("Error: %v", err)
				}
			}()
			if optClusterConfig == "" {
				stdout(fmt.Sprintf("No change. Input 'cfs-cli config set -h' for help.\n"))
				return
			}

			if rootConfig, err = ReadYamlConfig(optClusterConfig); err != nil {
				stdout(fmt.Sprintf("error %v\n", err))
			}

			clusterConfig := rootConfig.Cluster

			stdout(fmt.Sprintf("cluster: %v\n", clusterConfig))

			masterAddrs := make([]string, 0)
			masterConfig := clusterConfig.Master
			if masterConfig != nil {

				upStrIfEmpty(&masterConfig.LogDir, clusterConfig.LogDir)
				upStrIfEmpty(&masterConfig.LogLevel, clusterConfig.LogLevel)
				upStrIfEmpty(&masterConfig.ConsulAddr, clusterConfig.ConsulAddr)

				masterConfig.Peers = func() string {
					peers := make([]string, 0)
					for i, host := range masterConfig.Hosts {
						masterAddr := fmt.Sprintf("%s:%s", host, masterConfig.Listen)
						peers = append(peers, fmt.Sprintf("%d:%s", i, masterAddr))
					}
					return strings.Join(peers, ",")
				}()

				for i, host := range masterConfig.Hosts {
					masterConfig.ClusterName = clusterConfig.Name
					masterConfig.Id = fmt.Sprintf("%d", i)
					masterConfig.Ip = host

					masterAddr := fmt.Sprintf("%s:%s", host, masterConfig.Listen)
					masterAddrs = append(masterAddrs, masterAddr)

					master, _ := json.Marshal(masterConfig)
					stdout(fmt.Sprintf("master: %v\n", string(master)))
					if err = ioutil.WriteFile(fmt.Sprintf("%s/%s-%d.json", defConfigPath, "master", i), master, 0600); err != nil {
						return
					}
				}
			}

			metaNodeConfig := clusterConfig.MetaNode
			if metaNodeConfig != nil {
				metaNodeConfig.MasterAddr = masterAddrs

				upStrIfEmpty(&metaNodeConfig.LogDir, clusterConfig.LogDir)
				upStrIfEmpty(&metaNodeConfig.LogLevel, clusterConfig.LogLevel)
				upStrIfEmpty(&metaNodeConfig.WarnLogDir, clusterConfig.WarnLogDir)
				upStrIfEmpty(&metaNodeConfig.ConsulAddr, clusterConfig.ConsulAddr)

				conf, _ := json.Marshal(metaNodeConfig)
				stdout(fmt.Sprintf("meta: %v\n", string(conf)))
				if err = ioutil.WriteFile(defConfigPath+"/metanode.json", conf, 0600); err != nil {
					return
				}
			}

			dataNodeConfig := clusterConfig.DataNode
			if dataNodeConfig != nil {
				dataNodeConfig.MasterAddr = masterAddrs

				upStrIfEmpty(&dataNodeConfig.LogDir, clusterConfig.LogDir)
				upStrIfEmpty(&dataNodeConfig.WarnLogDir, clusterConfig.WarnLogDir)
				upStrIfEmpty(&dataNodeConfig.LogLevel, clusterConfig.LogLevel)
				upStrIfEmpty(&dataNodeConfig.ConsulAddr, clusterConfig.ConsulAddr)

				conf, _ := json.Marshal(dataNodeConfig)
				stdout(fmt.Sprintf("data: %v\n", string(conf)))
				if err = ioutil.WriteFile(defConfigPath+"/datanode.json", conf, 0600); err != nil {
					return
				}
			}

			clientConfig := clusterConfig.Client
			if clientConfig != nil {
				clientConfig.MasterAddr = strings.Join(masterAddrs, ",")

				upStrIfEmpty(&clientConfig.LogDir, clusterConfig.LogDir)
				upStrIfEmpty(&clientConfig.LogLevel, clusterConfig.LogLevel)
				upStrIfEmpty(&clientConfig.ConsulAddr, clusterConfig.ConsulAddr)

				confData, _ := json.Marshal(clientConfig)
				stdout(fmt.Sprintf("client: %v\n", string(confData)))
				if err = ioutil.WriteFile(defConfigPath+"/client.json", confData, 0600); err != nil {
					return
				}
			}

			//if err = setConfig(masterHosts, optTimeout); err != nil {
			//    return
			//}

			stdout(fmt.Sprintf("Config has been set successfully!\n"))
		},
	}
	cmd.Flags().StringVarP(&optClusterConfig, "config", "c", "", "Specify deploy cluster yaml file")
	//cmd.Flags().Uint16Var(&optTimeout, "timeout", 0, "Specify timeout for requests [Unit: s]")
	return cmd
}

func generateClusterConfig(cmd *cobra.Command, args []string) {
}
