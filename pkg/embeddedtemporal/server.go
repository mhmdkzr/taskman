package embeddedtemporal

import (
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"strconv"
	"time"

	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/server/common/authorization"
	"go.temporal.io/server/common/cluster"
	"go.temporal.io/server/common/config"
	"go.temporal.io/server/common/dynamicconfig"
	"go.temporal.io/server/common/log"
	"go.temporal.io/server/common/log/tag"
	"go.temporal.io/server/common/membership/static"
	"go.temporal.io/server/common/persistence/sql/sqlplugin/sqlite"
	sqliteplugin "go.temporal.io/server/common/persistence/sql/sqlplugin/sqlite"
	"go.temporal.io/server/common/primitives"
	sqliteschema "go.temporal.io/server/schema/sqlite"
	"go.temporal.io/server/temporal"
)

const localhost = "127.0.0.1"

type loggerAdapter struct {
	inner *slog.Logger
}

func (l loggerAdapter) Debug(msg string, tags ...tag.Tag) { l.log(slog.LevelDebug, msg, tags) }
func (l loggerAdapter) Info(msg string, tags ...tag.Tag)  { l.log(slog.LevelInfo, msg, tags) }
func (l loggerAdapter) Warn(msg string, tags ...tag.Tag)  { l.log(slog.LevelWarn, msg, tags) }
func (l loggerAdapter) Error(msg string, tags ...tag.Tag) { l.log(slog.LevelError, msg, tags) }
func (l loggerAdapter) DPanic(msg string, tags ...tag.Tag) { l.log(slog.LevelError, msg, tags) }
func (l loggerAdapter) Panic(msg string, tags ...tag.Tag)  { l.log(slog.LevelError, msg, tags) }
func (l loggerAdapter) Fatal(msg string, tags ...tag.Tag)  { l.log(slog.LevelError, msg, tags) }

func (l loggerAdapter) log(level slog.Level, msg string, tags []tag.Tag) {
	attrs := make([]slog.Attr, len(tags))
	for i, t := range tags {
		attrs[i] = slog.Any(t.Key(), t.Value())
	}
	l.inner.LogAttrs(nil, level, msg, attrs...)
}

// Server wraps an embedded Temporal server and its client.
type Server struct {
	temporalServer temporal.Server
	client         client.Client
	port           int
}

// Client returns the Temporal client connected to the embedded server.
func (s *Server) Client() client.Client { return s.client }

// Port returns the gRPC port the embedded server is listening on.
func (s *Server) Port() int { return s.port }

// Stop gracefully stops the embedded Temporal server.
func (s *Server) Stop() {
	if s.client != nil {
		s.client.Close()
	}
	if s.temporalServer != nil {
		s.temporalServer.Stop()
	}
}

// Start starts an embedded Temporal server with in-memory SQLite persistence
// and returns a handle for interacting with it.
func Start(namespace string) (*Server, error) {
	port, err := freePort()
	if err != nil {
		return nil, fmt.Errorf("find free port: %w", err)
	}

	cfg := buildConfig(port)

	sqlConf := &cfg.Persistence.DataStores["sqlite-default"].SQL

	if err := sqliteschema.SetupSchema(sqlConf); err != nil {
		return nil, fmt.Errorf("setup sqlite schema: %w", err)
	}

	nsConfig, err := sqlite.NewNamespaceConfig("active", namespace, false, nil)
	if err != nil {
		return nil, fmt.Errorf("create namespace config: %w", err)
	}
	if err := sqliteschema.CreateNamespaces(sqlConf, nsConfig); err != nil {
		return nil, fmt.Errorf("create namespace: %w", err)
	}

	slogLogger := slog.Default()
	logger := loggerAdapter{inner: slogLogger}

	authorizer, err := authorization.GetAuthorizerFromConfig(&cfg.Global.Authorization)
	if err != nil {
		return nil, fmt.Errorf("get authorizer: %w", err)
	}
	claimMapper, err := authorization.GetClaimMapperFromConfig(&cfg.Global.Authorization, logger)
	if err != nil {
		return nil, fmt.Errorf("get claim mapper: %w", err)
	}

	dynConf := make(dynamicconfig.StaticClient)
	dynConf[dynamicconfig.HistoryCacheHostLevelMaxSize.Key()] = 8096
	dynConf[dynamicconfig.FrontendMaxNamespaceVisibilityRPSPerInstance.Key()] = 100
	dynConf[dynamicconfig.EnableChasm.Key()] = true

	opts := []temporal.ServerOption{
		temporal.WithConfig(cfg),
		temporal.ForServices(temporal.DefaultServices),
		temporal.WithStaticHosts(map[primitives.ServiceName]static.Hosts{
			primitives.FrontendService: static.SingleLocalHost(
				fmt.Sprintf("%v:%v", localhost, cfg.Services[string(primitives.FrontendService)].RPC.GRPCPort)),
			primitives.MatchingService: static.SingleLocalHost(
				fmt.Sprintf("%v:%v", localhost, cfg.Services[string(primitives.MatchingService)].RPC.GRPCPort)),
			primitives.HistoryService: static.SingleLocalHost(
				fmt.Sprintf("%v:%v", localhost, cfg.Services[string(primitives.HistoryService)].RPC.GRPCPort)),
			primitives.WorkerService: static.SingleLocalHost(
				fmt.Sprintf("%v:%v", localhost, cfg.Services[string(primitives.WorkerService)].RPC.GRPCPort)),
		}),
		temporal.WithLogger(logger),
		temporal.WithAuthorizer(authorizer),
		temporal.WithClaimMapper(func(*config.Config) authorization.ClaimMapper { return claimMapper }),
		temporal.WithDynamicConfigClient(dynConf),
	}

	server, err := temporal.NewServer(opts...)
	if err != nil {
		return nil, fmt.Errorf("create temporal server: %w", err)
	}

	if err := server.Start(); err != nil {
		return nil, fmt.Errorf("start temporal server: %w", err)
	}

	cl, err := client.Dial(client.Options{
		HostPort:  fmt.Sprintf("%v:%v", localhost, port),
		Namespace: namespace,
		Logger:    slogLogger,
	})
	if err != nil {
		server.Stop()
		return nil, fmt.Errorf("dial temporal client: %w", err)
	}

	return &Server{temporalServer: server, client: cl, port: port}, nil
}

func buildConfig(port int) *config.Config {
	frontendPort := port
	matchingPort := port + 1
	historyPort := port + 2
	workerPort := port + 3

	dbName := strconv.Itoa(rand.Intn(9999999))

	return &config.Config{
		Global: config.Global{
			Membership: config.Membership{
				MaxJoinDuration: 30 * time.Second,
				BroadcastAddress: localhost,
			},
		},
		Persistence: config.Persistence{
			DefaultStore:     "sqlite-default",
			VisibilityStore:  "sqlite-default",
			NumHistoryShards: 1,
			DataStores: map[string]config.DataStore{
				"sqlite-default": {
					SQL: &config.SQL{
						PluginName: sqliteplugin.PluginName,
						ConnectAttributes: map[string]string{
							"mode":  "memory",
							"cache": "shared",
						},
						DatabaseName: dbName,
					},
				},
			},
		},
		ClusterMetadata: &cluster.Config{
			EnableGlobalNamespace:    false,
			FailoverVersionIncrement: 10,
			MasterClusterName:        "active",
			CurrentClusterName:       "active",
			ClusterInformation: map[string]cluster.ClusterInformation{
				"active": {
					Enabled:                true,
					InitialFailoverVersion: 1,
					RPCAddress:             fmt.Sprintf("%v:%v", localhost, frontendPort),
				},
			},
		},
		DCRedirectionPolicy: config.DCRedirectionPolicy{
			Policy: "noop",
		},
		Services: map[string]config.Service{
			"frontend": {
				RPC: config.RPC{
					GRPCPort:  frontendPort,
					BindOnIP:  localhost,
				},
			},
			"history": {
				RPC: config.RPC{
					GRPCPort:  historyPort,
					BindOnIP:  localhost,
				},
			},
			"matching": {
				RPC: config.RPC{
					GRPCPort:  matchingPort,
					BindOnIP:  localhost,
				},
			},
			"worker": {
				RPC: config.RPC{
					GRPCPort:  workerPort,
					BindOnIP:  localhost,
				},
			},
		},
		Archival: config.Archival{
			History: config.HistoryArchival{
				State: "disabled",
			},
			Visibility: config.VisibilityArchival{
				State: "disabled",
			},
		},
		NamespaceDefaults: config.NamespaceDefaults{
			Archival: config.ArchivalNamespaceDefaults{
				History: config.HistoryArchivalNamespaceDefaults{
					State: "disabled",
				},
				Visibility: config.VisibilityArchivalNamespaceDefaults{
					State: "disabled",
				},
			},
		},
		PublicClient: config.PublicClient{
			HostPort: fmt.Sprintf("%v:%v", localhost, frontendPort),
		},
	}
}

func freePort() (int, error) {
	l, err := net.Listen("tcp", localhost+":0")
	if err != nil {
		return 0, fmt.Errorf("failed to assign a free port: %w", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
