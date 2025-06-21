package config

type InfraConfig struct {
	Database Database
	Server   Server
	Proxy    Proxy
	GRPC     GRPC
}

type Database struct {
	BinPath string
	Host    string
	Port    string
	User    string
	Pass    string
	PoolMax int
}

type GRPC struct {
	Port string
	Host string
}

type Server struct {
	SRVPort string
}

type Proxy struct {
	PidPath  string
	MimePath string

	CertPath    string
	KeyPath     string
	DhparamPath string

	SRVPort   string
	HTTPPort  string
	HTTPSPort string

	AssetsPath string
	DistPath   string
}
