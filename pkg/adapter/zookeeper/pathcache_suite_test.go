package zookeeper

import (
	"context"
	"fmt"
	"github.com/go-zookeeper/zk"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/symcn/meshach/pkg/option"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"strings"
	"testing"
	"time"
)

const (
	_testConfigName   = "zoo.cfg"
	_testMyIDFileName = "myid"
)

const (
	DefaultServerTickTime                 = 500
	DefaultServerInitLimit                = 10
	DefaultServerSyncLimit                = 5
	DefaultServerAutoPurgeSnapRetainCount = 3
	DefaultPeerPort                       = 2888
	DefaultLeaderElectionPort             = 3888
	DefaultPort                           = 2181
)

type ErrMissingServerConfigField string

func (e ErrMissingServerConfigField) Error() string {
	return fmt.Sprintf("zk: missing server config field '%s'", string(e))
}

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "PathCache test suite", []Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter))
	By("Test case for the functions of PathCache")
	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("Tearing down")
})

type LogWriter struct {
	t *testing.T
	p string
}

var testOpt = &option.Registry{
	Type:    "zk",
	Address: []string{"10.12.214.41:2181"},
	Timeout: 30000,
}

type TestServer struct {
	Port   int
	Path   string
	Srv    *Server
	Config ServerConfigServer
}

type TestCluster struct {
	Path    string
	Config  ServerConfig
	Servers []TestServer
}

type ServerConfigServer struct {
	ID                 int
	Host               string
	PeerPort           int
	LeaderElectionPort int
}

type ServerConfig struct {
	TickTime                 int    // Number of milliseconds of each tick
	InitLimit                int    // Number of ticks that the initial synchronization phase can take
	SyncLimit                int    // Number of ticks that can pass between sending a request and getting an acknowledgement
	DataDir                  string // Direcrory where the snapshot is stored
	ClientPort               int    // Port at which clients will connect
	AutoPurgeSnapRetainCount int    // Number of snapshots to retain in dataDir
	AutoPurgePurgeInterval   int    // Purge task internal in hours (0 to disable auto purge)
	Servers                  []ServerConfigServer
}

type Server struct {
	stdout, stderr io.Writer
	cmdString      string
	cmdArgs        []string
	cmdEnv         []string
	cmd            *exec.Cmd
	// this cancel will kill the command being run in this case the server itself.
	cancelFunc context.CancelFunc
}

func (srv *Server) Start() error {
	ctx, cancel := context.WithCancel(context.Background())
	srv.cancelFunc = cancel

	srv.cmd = exec.CommandContext(ctx, srv.cmdString, srv.cmdArgs...)
	srv.cmd.Stdout = srv.stdout
	srv.cmd.Stderr = srv.stderr
	srv.cmd.Env = srv.cmdEnv

	return srv.cmd.Start()
}

func (srv *Server) Stop() error {
	srv.cancelFunc()
	return srv.cmd.Wait()
}

// TODO: pull this into its own package to allow for better isolation of integration tests vs. unit
// testing. This should be used on CI systems and local only when needed whereas unit tests should remain
// fast and not rely on external dependecies.
func StartTestCluster(size int, stdout, stderr io.Writer) (*TestCluster, error) {
	var err error
	tmpPath, err := ioutil.TempDir("", "gozk")

	success := false
	startPort := int(rand.Int31n(6000) + 10000)
	cluster := &TestCluster{Path: tmpPath}

	defer func() {
		if !success {
			cluster.Stop()
		}
	}()

	for serverN := 0; serverN < size; serverN++ {
		srvPath := filepath.Join(tmpPath, fmt.Sprintf("srv%d", serverN+1))

		port := startPort + serverN*3
		cfg := ServerConfig{
			ClientPort: port,
			DataDir:    srvPath,
		}

		for i := 0; i < size; i++ {
			serverNConfig := ServerConfigServer{
				ID:                 i + 1,
				Host:               "127.0.0.1",
				PeerPort:           startPort + i*3 + 1,
				LeaderElectionPort: startPort + i*3 + 2,
			}

			cfg.Servers = append(cfg.Servers, serverNConfig)
		}

		cfgPath := filepath.Join(srvPath, _testConfigName)
		fi, _ := os.Create(cfgPath)

		fi.Close()

		fi, err = os.Create(filepath.Join(srvPath, _testMyIDFileName))

		_, err = fmt.Fprintf(fi, "%d\n", serverN+1)
		fi.Close()

		srv, err := NewIntegrationTestServer(cfgPath, stdout, stderr)
		if err != nil {
			return nil, err
		}

		if err := srv.Start(); err != nil {
			return nil, err
		}

		cluster.Servers = append(cluster.Servers, TestServer{
			Path:   srvPath,
			Port:   cfg.ClientPort,
			Srv:    srv,
			Config: cfg.Servers[serverN],
		})
		cluster.Config = cfg
	}

	if err := cluster.waitForStart(30, time.Second); err != nil {
		return nil, err
	}

	success = true

	return cluster, err
}

func (tc *TestCluster) Connect(idx int) (*zk.Conn, <-chan zk.Event, error) {
	return zk.Connect([]string{fmt.Sprintf("127.0.0.1:%d", tc.Servers[idx].Port)}, time.Second*15)
}

func (tc *TestCluster) ConnectAll() (*zk.Conn, <-chan zk.Event, error) {
	return tc.ConnectAllTimeout(time.Second * 15)
}

func (tc *TestCluster) ConnectAllTimeout(sessionTimeout time.Duration) (*zk.Conn, <-chan zk.Event, error) {
	return tc.ConnectWithOptions(sessionTimeout)
}

func (tc *TestCluster) ConnectWithOptions(sessionTimeout time.Duration) (*zk.Conn, <-chan zk.Event, error) {
	hosts := make([]string, len(tc.Servers))
	for i, srv := range tc.Servers {
		hosts[i] = fmt.Sprintf("127.0.0.1:%d", srv.Port)
	}
	zk, ch, err := zk.Connect(hosts, sessionTimeout)
	return zk, ch, err
}

func (tc *TestCluster) Stop() error {
	for _, srv := range tc.Servers {
		srv.Srv.Stop()
	}
	defer os.RemoveAll(tc.Path)
	return tc.waitForStop(5, time.Second)
}

// waitForStart blocks until the cluster is up
func (tc *TestCluster) waitForStart(maxRetry int, interval time.Duration) error {
	// verify that the servers are up with SRVR
	serverAddrs := make([]string, len(tc.Servers))
	for i, s := range tc.Servers {
		serverAddrs[i] = fmt.Sprintf("127.0.0.1:%d", s.Port)
	}

	for i := 0; i < maxRetry; i++ {
		_, ok := zk.FLWSrvr(serverAddrs, time.Second)
		if ok {
			return nil
		}
		time.Sleep(interval)
	}

	return fmt.Errorf("unable to verify health of servers")
}

// waitForStop blocks until the cluster is down
func (tc *TestCluster) waitForStop(maxRetry int, interval time.Duration) error {
	// verify that the servers are up with RUOK
	serverAddrs := make([]string, len(tc.Servers))
	for i, s := range tc.Servers {
		serverAddrs[i] = fmt.Sprintf("127.0.0.1:%d", s.Port)
	}

	var success bool
	for i := 0; i < maxRetry && !success; i++ {
		success = true
		for _, ok := range zk.FLWRuok(serverAddrs, time.Second) {
			if ok {
				success = false
			}
		}
		if !success {
			time.Sleep(interval)
		}
	}
	if !success {
		return fmt.Errorf("unable to verify servers are down")
	}
	return nil
}

func (tc *TestCluster) StartServer(server string) {
	for _, s := range tc.Servers {
		if strings.HasSuffix(server, fmt.Sprintf(":%d", s.Port)) {
			s.Srv.Start()
			return
		}
	}
	panic(fmt.Sprintf("unknown server: %s", server))
}

func (tc *TestCluster) StopServer(server string) {
	for _, s := range tc.Servers {
		if strings.HasSuffix(server, fmt.Sprintf(":%d", s.Port)) {
			s.Srv.Stop()
			return
		}
	}
	panic(fmt.Sprintf("unknown server: %s", server))
}

func (tc *TestCluster) StartAllServers() error {
	for _, s := range tc.Servers {
		if err := s.Srv.Start(); err != nil {
			return fmt.Errorf("failed to start server listening on port `%d` : %+v", s.Port, err)
		}
	}

	if err := tc.waitForStart(10, time.Second*2); err != nil {
		return fmt.Errorf("failed to wait to startup zk servers: %v", err)
	}

	return nil
}

func (tc *TestCluster) StopAllServers() error {
	var err error
	for _, s := range tc.Servers {
		if err := s.Srv.Stop(); err != nil {
			err = fmt.Errorf("failed to stop server listening on port `%d` : %v", s.Port, err)
		}
	}
	if err != nil {
		return err
	}

	if err := tc.waitForStop(5, time.Second); err != nil {
		return fmt.Errorf("failed to wait to startup zk servers: %v", err)
	}

	return nil
}

func requireNoError(t *testing.T, err error, msgAndArgs ...interface{}) {
	if err != nil {
		t.Logf("received unexpected error: %v", err)
		t.Fatal(msgAndArgs...)
	}
}

// nolint: golint
func NewIntegrationTestServer(configPath string, stdout, stderr io.Writer) (*Server, error) {
	// allow external systems to configure this zk server bin path.
	zkPath := os.Getenv("ZOOKEEPER_BIN_PATH")
	if zkPath == "" {
		// default to a static reletive path that can be setup with a build system
		zkPath = "zookeeper/bin"
	}
	if _, err := os.Stat(zkPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("zk: could not find testing zookeeper bin path at %q: %v ", zkPath, err)
		}
	}
	// password is 'test'
	superString := `SERVER_JVMFLAGS=-Dzookeeper.DigestAuthenticationProvider.superDigest=super:D/InIHSb7yEEbrWz8b9l71RjZJU=`
	// enable TTL
	superString += ` -Dzookeeper.extendedTypesEnabled=true -Dzookeeper.emulate353TTLNodes=true`

	return &Server{
		cmdString: filepath.Join(zkPath, "zkServer.sh"),
		cmdArgs:   []string{"start-foreground", configPath},
		cmdEnv:    []string{superString},
		stdout:    stdout, stderr: stderr,
	}, nil
}

func (sc ServerConfig) Marshall(w io.Writer) error {
	// the admin server is not wanted in test cases as it slows the startup process and is
	// of little unit test value.
	fmt.Fprintln(w, "admin.enableServer=false")
	if sc.DataDir == "" {
		return ErrMissingServerConfigField("dataDir")
	}
	fmt.Fprintf(w, "dataDir=%s\n", sc.DataDir)
	if sc.TickTime <= 0 {
		sc.TickTime = DefaultServerTickTime
	}
	fmt.Fprintf(w, "tickTime=%d\n", sc.TickTime)
	if sc.InitLimit <= 0 {
		sc.InitLimit = DefaultServerInitLimit
	}
	fmt.Fprintf(w, "initLimit=%d\n", sc.InitLimit)
	if sc.SyncLimit <= 0 {
		sc.SyncLimit = DefaultServerSyncLimit
	}
	fmt.Fprintf(w, "syncLimit=%d\n", sc.SyncLimit)
	if sc.ClientPort <= 0 {
		sc.ClientPort = DefaultPort
	}
	fmt.Fprintf(w, "clientPort=%d\n", sc.ClientPort)
	if sc.AutoPurgePurgeInterval > 0 {
		if sc.AutoPurgeSnapRetainCount <= 0 {
			sc.AutoPurgeSnapRetainCount = DefaultServerAutoPurgeSnapRetainCount
		}
		fmt.Fprintf(w, "autopurge.snapRetainCount=%d\n", sc.AutoPurgeSnapRetainCount)
		fmt.Fprintf(w, "autopurge.purgeInterval=%d\n", sc.AutoPurgePurgeInterval)
	}
	// enable reconfig.
	// TODO: allow setting this
	fmt.Fprintln(w, "reconfigEnabled=true")
	fmt.Fprintln(w, "4lw.commands.whitelist=*")

	if len(sc.Servers) < 2 {
		// if we dont have more than 2 servers we just dont specify server list to start in standalone mode
		// see https://zookeeper.apache.org/doc/current/zookeeperStarted.html#sc_InstallingSingleMode for more details.
		return nil
	}
	// if we then have more than one server force it to be distributed
	fmt.Fprintln(w, "standaloneEnabled=false")

	for _, srv := range sc.Servers {
		if srv.PeerPort <= 0 {
			srv.PeerPort = DefaultPeerPort
		}
		if srv.LeaderElectionPort <= 0 {
			srv.LeaderElectionPort = DefaultLeaderElectionPort
		}
		fmt.Fprintf(w, "server.%d=%s:%d:%d\n", srv.ID, srv.Host, srv.PeerPort, srv.LeaderElectionPort)
	}
	return nil
}
