package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/shilkin/centrifugo/Godeps/_workspace/src/github.com/spf13/cobra"
	"github.com/shilkin/centrifugo/Godeps/_workspace/src/github.com/spf13/viper"
	"github.com/shilkin/centrifugo/libcentrifugo"
	"github.com/shilkin/centrifugo/libcentrifugo/logger"
	"github.com/shilkin/go-tarantool"
)

const (
	VERSION = "0.2.4"
)

var configFile string

func setupLogging() {
	logLevel, ok := logger.LevelMatches[strings.ToUpper(viper.GetString("log_level"))]
	if !ok {
		logLevel = logger.LevelInfo
	}
	logger.SetLogThreshold(logLevel)
	logger.SetStdoutThreshold(logLevel)

	if viper.IsSet("log_file") && viper.GetString("log_file") != "" {
		logger.SetLogFile(viper.GetString("log_file"))
		// do not log into stdout when log file provided
		logger.SetStdoutThreshold(logger.LevelNone)
	}
}

func handleSignals(app *libcentrifugo.Application) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, os.Interrupt)
	for {
		sig := <-sigc
		logger.INFO.Println("Signal received:", sig)
		switch sig {
		case syscall.SIGHUP:
			// reload application configuration on SIGHUP
			logger.INFO.Println("Reloading configuration")
			err := viper.ReadInConfig()
			if err != nil {
				switch err.(type) {
				case viper.ConfigParseError:
					logger.CRITICAL.Printf("Error parsing configuration: %s\n", err)
					continue
				default:
					logger.CRITICAL.Println("Unable to locate config file")
					continue
				}
			}
			setupLogging()
			c := newConfig()
			s := structureFromConfig(nil)
			app.SetConfig(c)
			app.SetStructure(s)
			logger.INFO.Println("Configuration successfully reloaded")
		case syscall.SIGINT, os.Interrupt:
			logger.INFO.Println("Shutting down")
			go time.AfterFunc(5*time.Second, func() {
				os.Exit(1)
			})
			app.Shutdown()
			os.Exit(130)
		}
	}
}

// Main runs Centrifugo as a service.
func Main() {

	var port string
	var address string
	var debug bool
	var name string
	var web string
	var engn string
	var logLevel string
	var logFile string
	var insecure bool
	var useSSL bool
	var sslCert string
	var sslKey string

	var redisHost string
	var redisPort string
	var redisPassword string
	var redisDB string
	var redisURL string
	var redisAPI bool
	var redisPool int

	// Tarantool
	var ttHost string
	var ttPort string
	var ttUser string
	var ttPassword string
	var ttPool int
	var ttTimeoutResponse int
	var ttTimeoutReconnect int
	var ttMaxReconnect uint

	var rootCmd = &cobra.Command{
		Use:   "",
		Short: "Centrifugo",
		Long:  "Centrifuge + GO = Centrifugo – harder, better, faster, stronger",
		Run: func(cmd *cobra.Command, args []string) {

			viper.SetDefault("gomaxprocs", 0)
			viper.SetDefault("prefix", "")
			viper.SetDefault("web_password", "")
			viper.SetDefault("web_secret", "")
			viper.RegisterAlias("cookie_secret", "web_secret")
			viper.SetDefault("max_channel_length", 255)
			viper.SetDefault("channel_prefix", "centrifugo")
			viper.SetDefault("node_ping_interval", 5)
			viper.SetDefault("message_send_timeout", 60)
			viper.SetDefault("expired_connection_close_delay", 10)
			viper.SetDefault("presence_ping_interval", 25)
			viper.SetDefault("presence_expire_interval", 60)
			viper.SetDefault("private_channel_prefix", "$")
			viper.SetDefault("namespace_channel_boundary", ":")
			viper.SetDefault("user_channel_boundary", "#")
			viper.SetDefault("user_channel_separator", ",")
			viper.SetDefault("client_channel_boundary", "&")
			viper.SetDefault("sockjs_url", "https://cdn.jsdelivr.net/sockjs/1.0/sockjs.min.js")

			viper.SetDefault("project_name", "")
			viper.SetDefault("project_secret", "")
			viper.SetDefault("project_connection_lifetime", false)
			viper.SetDefault("project_watch", false)
			viper.SetDefault("project_publish", false)
			viper.SetDefault("project_anonymous", false)
			viper.SetDefault("project_presence", false)
			viper.SetDefault("project_history_size", 0)
			viper.SetDefault("project_history_lifetime", 0)
			viper.SetDefault("project_namespaces", "")

			// Tarantool defaults
			/*
				viper.SetDefault("tt_pool", 2)
				viper.SetDefault("tt_host", "127.0.0.1")
				viper.SetDefault("tt_port", "3301")
				viper.SetDefault("tt_user", "")
				viper.SetDefault("tt_password", "")
				viper.SetDefault("tt_timeout_response", 500)
				viper.SetDefault("tt_timeout_reconnect", 500)
			*/

			viper.SetEnvPrefix("centrifugo")
			viper.BindEnv("engine")
			viper.BindEnv("insecure")
			viper.BindEnv("web_password")
			viper.BindEnv("web_secret")
			viper.BindEnv("project_name")
			viper.BindEnv("project_secret")
			viper.BindEnv("project_connection_lifetime")
			viper.BindEnv("project_watch")
			viper.BindEnv("project_publish")
			viper.BindEnv("project_anonymous")
			viper.BindEnv("project_join_leave")
			viper.BindEnv("project_presence")
			viper.BindEnv("project_history_size")
			viper.BindEnv("project_history_lifetime")

			viper.BindPFlag("port", cmd.Flags().Lookup("port"))
			viper.BindPFlag("address", cmd.Flags().Lookup("address"))
			viper.BindPFlag("debug", cmd.Flags().Lookup("debug"))
			viper.BindPFlag("name", cmd.Flags().Lookup("name"))
			viper.BindPFlag("web", cmd.Flags().Lookup("web"))
			viper.BindPFlag("engine", cmd.Flags().Lookup("engine"))
			viper.BindPFlag("insecure", cmd.Flags().Lookup("insecure"))
			viper.BindPFlag("ssl", cmd.Flags().Lookup("ssl"))
			viper.BindPFlag("ssl_cert", cmd.Flags().Lookup("ssl_cert"))
			viper.BindPFlag("ssl_key", cmd.Flags().Lookup("ssl_key"))
			viper.BindPFlag("log_level", cmd.Flags().Lookup("log_level"))
			viper.BindPFlag("log_file", cmd.Flags().Lookup("log_file"))
			viper.BindPFlag("redis_host", cmd.Flags().Lookup("redis_host"))
			viper.BindPFlag("redis_port", cmd.Flags().Lookup("redis_port"))
			viper.BindPFlag("redis_password", cmd.Flags().Lookup("redis_password"))
			viper.BindPFlag("redis_db", cmd.Flags().Lookup("redis_db"))
			viper.BindPFlag("redis_url", cmd.Flags().Lookup("redis_url"))
			viper.BindPFlag("redis_api", cmd.Flags().Lookup("redis_api"))
			viper.BindPFlag("redis_pool", cmd.Flags().Lookup("redis_pool"))

			// Tarantool
			viper.BindPFlag("tt_pool", cmd.Flags().Lookup("tt_pool"))
			viper.BindPFlag("tt_host", cmd.Flags().Lookup("tt_host"))
			viper.BindPFlag("tt_port", cmd.Flags().Lookup("tt_port"))
			viper.BindPFlag("tt_user", cmd.Flags().Lookup("tt_user"))
			viper.BindPFlag("tt_password", cmd.Flags().Lookup("tt_password"))
			viper.BindPFlag("tt_timeout_response", cmd.Flags().Lookup("tt_timeout_request"))
			viper.BindPFlag("tt_timeout_reconnect", cmd.Flags().Lookup("tt_timeout_reconnect"))
			viper.BindPFlag("tt_max_reconnect", cmd.Flags().Lookup("tt_max_reconnect"))

			err := validateConfig(configFile)
			if err != nil {
				logger.FATAL.Fatalln(err)
			}

			viper.SetConfigFile(configFile)
			err = viper.ReadInConfig()
			if err != nil {
				logger.FATAL.Fatalln("Unable to locate config file")
			}
			setupLogging()

			if os.Getenv("GOMAXPROCS") == "" {
				if viper.IsSet("gomaxprocs") && viper.GetInt("gomaxprocs") > 0 {
					runtime.GOMAXPROCS(viper.GetInt("gomaxprocs"))
				} else {
					runtime.GOMAXPROCS(runtime.NumCPU())
				}
			}

			logger.INFO.Println("GOMAXPROCS set to", runtime.GOMAXPROCS(0))
			logger.INFO.Println("Using config file:", viper.ConfigFileUsed())

			c := newConfig()
			s := structureFromConfig(nil)

			app, err := libcentrifugo.NewApplication(c)
			if err != nil {
				logger.FATAL.Fatalln(err)
			}

			app.SetStructure(s)

			var e libcentrifugo.Engine
			switch viper.GetString("engine") {
			case "memory":
				e = libcentrifugo.NewMemoryEngine(app)
			case "redis":
				e = libcentrifugo.NewRedisEngine(
					app,
					viper.GetString("redis_host"),
					viper.GetString("redis_port"),
					viper.GetString("redis_password"),
					viper.GetString("redis_db"),
					viper.GetString("redis_url"),
					viper.GetBool("redis_api"),
					viper.GetInt("redis_pool"),
				)
			case "tarantool":
				hostname, err := os.Hostname()
				if err != nil {
					logger.FATAL.Fatalln("Unable to determine local hostname: " + err.Error())
				}
				config := libcentrifugo.TarantoolEngineConfig{
					PoolConfig: libcentrifugo.TarantoolPoolConfig{
						Address:  viper.GetString("tt_host") + ":" + viper.GetString("tt_port"),
						PoolSize: viper.GetInt("tt_pool"),
						Opts: tarantool.Opts{
							time.Duration(viper.GetInt("tt_timeout_response")) * time.Millisecond,
							time.Duration(viper.GetInt("tt_timeout_reconnect")) * time.Millisecond,
							viper.GetString("tt_user"),
							viper.GetString("tt_password"),
						},
					},
					Endpoint: fmt.Sprintf("http://%s:%s", hostname, viper.GetString("port")),
				}
				e = libcentrifugo.NewTarantoolEngine(app, config)
			default:
				logger.FATAL.Fatalln("unknown engine: " + viper.GetString("engine"))
			}

			logger.INFO.Println("Engine:", viper.GetString("engine"))
			logger.DEBUG.Printf("%v\n", viper.AllSettings())
			logger.INFO.Println("Use SSL:", viper.GetBool("ssl"))
			if viper.GetBool("ssl") {
				if viper.GetString("ssl_cert") == "" {
					logger.FATAL.Println("No SSL certificate provided")
					os.Exit(1)
				}
				if viper.GetString("ssl_key") == "" {
					logger.FATAL.Println("No SSL certificate key provided")
					os.Exit(1)
				}
			}
			app.SetEngine(e)

			app.Run()

			go handleSignals(app)

			mux := libcentrifugo.DefaultMux(app, viper.GetString("prefix"), viper.GetString("web"), viper.GetString("sockjs_url"))

			addr := viper.GetString("address") + ":" + viper.GetString("port")
			logger.INFO.Printf("Start serving on %s\n", addr)
			if useSSL {
				if err := http.ListenAndServeTLS(addr, sslCert, sslKey, mux); err != nil {
					logger.FATAL.Fatalln("ListenAndServe:", err)
				}
			} else {
				if err := http.ListenAndServe(addr, mux); err != nil {
					logger.FATAL.Fatalln("ListenAndServe:", err)
				}
			}
		},
	}
	rootCmd.Flags().StringVarP(&port, "port", "p", "8000", "port to bind to")
	rootCmd.Flags().StringVarP(&address, "address", "a", "", "address to listen on")
	rootCmd.Flags().BoolVarP(&debug, "debug", "d", false, "debug mode - please, do not use it in production")
	rootCmd.Flags().StringVarP(&configFile, "config", "c", "config.json", "path to config file")
	rootCmd.Flags().StringVarP(&name, "name", "n", "", "unique node name")
	rootCmd.Flags().StringVarP(&web, "web", "w", "", "optional path to web interface application")
	rootCmd.Flags().StringVarP(&engn, "engine", "e", "memory", "engine to use: memory, redis, tarantool")
	rootCmd.Flags().BoolVarP(&insecure, "insecure", "", false, "start in insecure mode")
	rootCmd.Flags().BoolVarP(&useSSL, "ssl", "", false, "accept SSL connections. This requires an X509 certificate and a key file")
	rootCmd.Flags().StringVarP(&sslCert, "ssl_cert", "", "", "path to an X509 certificate file")
	rootCmd.Flags().StringVarP(&sslKey, "ssl_key", "", "", "path to an X509 certificate key")
	rootCmd.Flags().StringVarP(&logLevel, "log_level", "", "info", "set the log level: debug, info, error, critical, fatal or none")
	rootCmd.Flags().StringVarP(&logFile, "log_file", "", "", "optional log file - if not specified all logs go to STDOUT")
	rootCmd.Flags().StringVarP(&redisHost, "redis_host", "", "127.0.0.1", "redis host (Redis engine)")
	rootCmd.Flags().StringVarP(&redisPort, "redis_port", "", "6379", "redis port (Redis engine)")
	rootCmd.Flags().StringVarP(&redisPassword, "redis_password", "", "", "redis auth password (Redis engine)")
	rootCmd.Flags().StringVarP(&redisDB, "redis_db", "", "0", "redis database (Redis engine)")
	rootCmd.Flags().StringVarP(&redisURL, "redis_url", "", "", "redis connection URL (Redis engine)")
	rootCmd.Flags().BoolVarP(&redisAPI, "redis_api", "", false, "enable Redis API listener (Redis engine)")
	rootCmd.Flags().IntVarP(&redisPool, "redis_pool", "", 256, "Redis pool size (Redis engine)")

	// Tarantool
	rootCmd.Flags().StringVarP(&ttHost, "tt_host", "", "127.0.0.1", "tarantool host (Tarantool engine)")
	rootCmd.Flags().StringVarP(&ttPort, "tt_port", "", "3301", "tarantool port (Tarantool engine)")
	rootCmd.Flags().StringVarP(&ttUser, "tt_user", "", "", "tarantool user (Tarantool engine)")
	rootCmd.Flags().StringVarP(&ttPassword, "tt_password", "", "", "tarantool password (Tarantool engine)")
	rootCmd.Flags().IntVarP(&ttPool, "tt_pool", "", 2, "tarantool connection pool size (Tarantool engine)")
	rootCmd.Flags().IntVarP(&ttTimeoutResponse, "tt_timeout_response", "", 500, "timeout to wait response in milliseconds (Tarantool engine)")
	rootCmd.Flags().IntVarP(&ttTimeoutReconnect, "tt_timeout_reconnect", "", 500, "timeout to wait until reconnection attempt in milliseconds (Tarantool engine)")
	rootCmd.Flags().UintVarP(&ttMaxReconnect, "tt_max_reconnect", "", ^uint(0), "max number of reconnection attempts (Tarantool engine)")

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Centrifugo version number",
		Long:  `Print the version number of Centrifugo`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Centrifugo v%s\n", VERSION)
		},
	}

	var checkConfigFile string

	var checkConfigCmd = &cobra.Command{
		Use:   "checkconfig",
		Short: "Check configuration file",
		Long:  `Check Centrifugo configuration file`,
		Run: func(cmd *cobra.Command, args []string) {
			err := validateConfig(checkConfigFile)
			if err != nil {
				logger.FATAL.Fatalln(err)
			}
		},
	}
	checkConfigCmd.Flags().StringVarP(&checkConfigFile, "config", "c", "config.json", "path to config file to check")

	var outputConfigFile string

	var generateConfigCmd = &cobra.Command{
		Use:   "genconfig",
		Short: "Generate simple configuration file to start with",
		Long:  `Generate simple configuration file to start with`,
		Run: func(cmd *cobra.Command, args []string) {
			err := generateConfig(outputConfigFile)
			if err != nil {
				logger.FATAL.Fatalln(err)
			}
		},
	}
	generateConfigCmd.Flags().StringVarP(&outputConfigFile, "config", "c", "config.json", "path to output config file")

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(checkConfigCmd)
	rootCmd.AddCommand(generateConfigCmd)
	rootCmd.Execute()
}

func main() {
	Main()
}
