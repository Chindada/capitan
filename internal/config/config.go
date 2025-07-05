// Package config package config
package config

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"text/template"
	"time"

	"github.com/chindada/capitan/internal/config/templates"
	gRPCClient "github.com/chindada/capitan/internal/usecases/grpc/client"
	"github.com/chindada/leopard/pkg/command"
	"github.com/chindada/leopard/pkg/log"
	"github.com/chindada/panther/golang/pb"
	"github.com/chindada/panther/pkg/client"
	"github.com/chindada/panther/pkg/launcher"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

const (
	dbName = "capitan"
)

// Config -.
type Config struct {
	InfraConfig

	vp      *viper.Viper
	logger  *log.Log
	gRPConn *grpc.ClientConn

	dbClient client.PGClient

	rootPath  string
	dbStarted bool
}

var (
	singleton *Config
	once      sync.Once
)

func newConfig() *Config {
	logger := log.Get()
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	return &Config{
		logger:   logger,
		vp:       viper.New(),
		rootPath: filepath.Join(filepath.Dir(ex), ".."),
	}
}

func (c *Config) loadEnv() {
	c.vp.SetDefault("DB_BIN_PATH", "")
	c.vp.SetDefault("DB_HOST", "127.0.0.1")
	c.vp.SetDefault("DB_PORT", "5432")
	c.vp.SetDefault("DB_USER", "postgres")
	c.vp.SetDefault("DB_PASS", "password")
	c.vp.SetDefault("DB_POOL_MAX", 90)
	c.vp.SetDefault("SRV_PORT", "23456")
	c.vp.SetDefault("HTTP_PORT", "80")
	c.vp.SetDefault("GRPC_PORT", "56666")
	c.vp.SetDefault("GRPC_HOST", "127.0.0.1")
	c.vp.AutomaticEnv()
	c.InfraConfig = InfraConfig{
		Database: Database{
			BinPath: c.vp.GetString("DB_BIN_PATH"),
			Host:    c.vp.GetString("DB_HOST"),
			Port:    c.vp.GetString("DB_PORT"),
			User:    c.vp.GetString("DB_USER"),
			Pass:    c.vp.GetString("DB_PASS"),
			PoolMax: c.vp.GetInt("DB_POOL_MAX"),
		},
		Server: Server{
			SRVPort: c.vp.GetString("SRV_PORT"),
		},
		GRPC: GRPC{
			Port: c.vp.GetString("GRPC_PORT"),
			Host: c.vp.GetString("GRPC_HOST"),
		},
		Proxy: Proxy{
			PidPath:    filepath.Join(c.rootPath, "proxy", "proxy.pid"),
			MimePath:   filepath.Join(c.rootPath, "proxy", "conf", "mime.types"),
			SRVPort:    c.vp.GetString("SRV_PORT"),
			HTTPPort:   c.vp.GetString("HTTP_PORT"),
			AssetsPath: filepath.Join(c.rootPath, "dist", "assets"),
			DistPath:   filepath.Join(c.rootPath, "dist"),
		},
	}
}

func (c *Config) writeProxyConfig() {
	var b bytes.Buffer
	t := template.Must(template.ParseFS(templates.Porxy, "proxy.tmpl"))
	err := t.Execute(&b, c.Proxy)
	if err != nil {
		c.logger.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(c.rootPath, "proxy", "conf", "nginx.conf"), b.Bytes(), 0o600)
	if err != nil {
		c.logger.Fatal(err)
	}
}

func (c *Config) StartProxy() {
	cmd := command.NewCMD(filepath.Join(c.rootPath, "proxy", "sbin", "proxy"))
	cmd.Dir = filepath.Join(c.rootPath, "proxy")
	err := cmd.Start()
	if err != nil {
		c.logger.Fatal(err)
	}
}

func (c *Config) StopProxy() {
	proxyPID, err := os.ReadFile(c.Proxy.PidPath)
	if err != nil {
		return
	}
	proxyPIDInt, err := strconv.Atoi(strings.ReplaceAll(string(proxyPID), "\n", ""))
	if err != nil {
		return
	}
	p, e := os.FindProcess(proxyPIDInt)
	if e != nil {
		return
	}
	e = p.Signal(syscall.SIGQUIT)
	if e != nil {
		return
	}
}

func Init() {
	once.Do(func() {
		c := newConfig()
		c.loadEnv()
		c.launchDB()
		c.setPostgresPool()
		c.writeProxyConfig()
		singleton = c
	})
}

func Get() *Config {
	if singleton == nil {
		once.Do(Init)
		return Get()
	}
	return singleton
}

func (c *Config) launchDB() {
	launcher.Init(
		launcher.Listen(c.Database.Host),
		launcher.Port(c.Database.Port),
		launcher.DBName(dbName),
		launcher.AddLogger(c.logger),
		launcher.BinaryRoot(c.Database.BinPath),
	)
	dbt := launcher.Get()
	defer c.runExporter(dbt)
	if !dbt.DatabaseAlreadyExists() {
		err := dbt.InitDB(true)
		if err != nil {
			c.logger.Fatal(err)
		}
	}
	if isRunning, err := dbt.IsRunning(); err != nil {
		c.logger.Fatal(err)
	} else if !isRunning {
		if err = dbt.StartDB(); err != nil {
			c.logger.Fatal(err)
		}
		c.dbStarted = true
	}
	if err := dbt.MigrateScheme(nil); err != nil {
		c.logger.Fatal(err)
	}
}

func (c *Config) setPostgresPool() {
	var path string
	dbt := launcher.Get()
	if socketPath := dbt.GetSocketPath(); socketPath != "" {
		path = fmt.Sprintf("postgres://%s:%s@?host=%s&port=%s&dbname=%s&sslmode=disable",
			c.Database.User, c.Database.Pass,
			socketPath, c.Database.Port, dbName)
		c.logger.Infof("database socket path: %s", socketPath)
	} else {
		path = fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable",
			c.Database.User, c.Database.Pass,
			net.JoinHostPort(c.Database.Host, c.Database.Port), dbName)
		c.logger.Infof("database host: %s", c.Database.Host)
	}
	pg, err := client.New(
		path,
		client.MaxPoolSize(c.Database.PoolMax),
		client.AddLogger(c.logger),
	)
	if err != nil {
		c.logger.Fatal(err)
	}
	c.dbClient = pg
}

func (c *Config) runExporter(dbt launcher.PGLauncher) {
	needExport, ok := os.LookupEnv("DB_EXPORTER")
	if ok {
		need, err := strconv.ParseBool(needExport)
		if err != nil {
			c.logger.Warn(err)
		} else if need {
			if err = dbt.RunExporter(); err != nil {
				c.logger.Warn(err)
			}
		}
	}
}

func (c *Config) ConnectGRPC(interrupt chan os.Signal) {
	if c.InfraConfig.GRPC.Host == "" {
		c.logger.Fatal("GRPC host is not set")
	}
	if c.InfraConfig.GRPC.Port == "" {
		c.logger.Fatal("GRPC port is not set")
	}
	retry := 60
	c.logger.Infof("Connecting to %s...", net.JoinHostPort(c.InfraConfig.GRPC.Host, c.InfraConfig.GRPC.Port))
	for i := range retry {
		if c.tryConnectGRPC(interrupt) {
			return
		}
		if i < retry-1 {
			c.logger.Infof("Waiting 3 second before next retry")
			<-time.After(time.Second * 3)
		}
	}
	if c.gRPConn == nil {
		c.logger.Fatalf("Failed to connect to gRPC server %s:%s after %d retries",
			c.InfraConfig.GRPC.Host, c.InfraConfig.GRPC.Port, retry)
	}
}

func (c *Config) tryConnectGRPC(interrupt chan os.Signal) bool {
	gRPConn, err := gRPCClient.NewInsecureClient(net.JoinHostPort(c.InfraConfig.GRPC.Host, c.InfraConfig.GRPC.Port))
	if err != nil {
		return false
	}
	healthClient := pb.NewHealthInterfaceClient(gRPConn)
	stream, cErr := healthClient.HealthChannel(context.Background())
	if cErr != nil {
		return false
	}
	go func() {
		for {
			_, err = stream.Recv()
			if err != nil {
				c.logger.Error("Lost connection to gRPC server")
				interrupt <- os.Interrupt
				return
			}
		}
	}()
	c.gRPConn = gRPConn
	c.logger.Info("Connected")
	return true
}

func (c *Config) GetGRPCConn() *grpc.ClientConn {
	if c.gRPConn == nil {
		c.logger.Fatal("gRPC not connected")
	}
	return c.gRPConn
}

func (c *Config) GetPostgresPool() client.PGClient {
	if c.dbClient == nil {
		c.logger.Fatal("postgres not connected")
	}
	return c.dbClient
}

func (c *Config) CloseDB() {
	if c.dbClient != nil {
		c.dbClient.Close()
	}
	if !c.dbStarted {
		return
	}
	dbt := launcher.Get()
	if err := dbt.StopDB(); err != nil {
		log.Get().Fatal(err)
	}
}
