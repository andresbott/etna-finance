package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/andresbott/etna/app/metainfo"
	"github.com/andresbott/etna/app/router"
	handlers "github.com/andresbott/etna/app/router/handlers"
	"github.com/andresbott/etna/app/tasks"
	"github.com/andresbott/etna/internal/accounting"
	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/taskrunner"
	"github.com/glebarez/sqlite"
	"github.com/go-bumbu/tempo"
	"github.com/go-bumbu/userauth"
	"github.com/go-bumbu/userauth/handlers/sessionauth"
	"github.com/go-bumbu/userauth/userstore/staticusers"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const dbFile = "carbon.db"
const sessionsDir = "sessions"
const backupsDir = "backup"

// buildTaskRegisterFuncs returns the map of task name -> register func for the given runner and stores.
// When production is true, dev-only tasks (log-only, log-only-long) are not included.
func buildTaskRegisterFuncs(
	runner *taskrunner.Runner,
	finStore *accounting.Store,
	marketStore *marketdata.Store,
	backupDestination string,
	logger *slog.Logger,
	production bool,
) map[string]func() (uuid.UUID, error) {
	if runner == nil {
		return nil
	}
	registerFuncs := map[string]func() (uuid.UUID, error){
		tasks.BackupTaskName: func() (uuid.UUID, error) {
			return runner.RegisterTask(
				tasks.NewBackupTaskFn(finStore, backupDestination, logger),
				tasks.BackupTaskName,
			)
		},
		tasks.FinancialImportTaskName: func() (uuid.UUID, error) {
			return runner.RegisterTask(
				tasks.NewFinancialImportTaskFn(marketStore, logger),
				tasks.FinancialImportTaskName,
			)
		},
	}
	if !production {
		registerFuncs[tasks.LogOnlyTaskName] = func() (uuid.UUID, error) {
			return runner.RegisterTaskWithMaxParallelism(
				tasks.NewLogOnlyTaskFn(logger),
				tasks.LogOnlyTaskName,
				4, // short demo task: up to 4 concurrent
			)
		}
		registerFuncs[tasks.LogOnlyLongTaskName] = func() (uuid.UUID, error) {
			return runner.RegisterTaskWithMaxParallelism(
				tasks.NewLogOnlyLongTaskFn(logger),
				tasks.LogOnlyLongTaskName,
				1, // long demo task: only 1 at a time
			)
		}
		registerFuncs[tasks.DebugFailTaskName] = func() (uuid.UUID, error) {
			return runner.RegisterTask(
				tasks.NewDebugFailTaskFn(logger),
				tasks.DebugFailTaskName,
			)
		}
	}
	return registerFuncs
}

func serverCmd() *cobra.Command {
	var configFile = "./config.yaml"
	cmd := &cobra.Command{
		Use:   "start",
		Short: "start a web server",
		Long:  "start a web server demonstrating the different features of the library",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runServer(configFile)
		},
	}

	cmd.Flags().StringVarP(&configFile, "config", "c", configFile, "config file")
	return cmd
}

func runServer(configFile string) error {
	// ——— Config and logger ———
	cfg, err := getAppCfg(configFile)
	if err != nil {
		return err
	}
	_ = cfg
	l, err := defaultLogger(GetLogLevel(cfg.Env.LogLevel))
	if err != nil {
		return err
	}

	l.Info("App startup",
		slog.String("component", "startup"),
		slog.String("version", metainfo.Version),
		slog.String("Build Date", metainfo.BuildTime),
		slog.String("commit", metainfo.ShaVer),
	)
	for _, m := range cfg.Msgs {
		if m.Level == "info" {
			l.Info(m.Msg, slog.String("component", "config"))
		} else {
			l.Debug(m.Msg, slog.String("component", "config"))
		}
	}

	// ——— Data directory ———
	err = initDataDir(cfg.DataDir)
	if err != nil {
		return err
	}
	l.Info("using data directory", slog.String("path", cfg.DataDir))

	// ——— Database ———
	gormLogger := gormlogger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		gormlogger.Config{
			IgnoreRecordNotFoundError: true,
			LogLevel:                  gormlogger.Warn, // only warnings and errors, not every SQL
		},
	)
	db, err := gorm.Open(sqlite.Open(filepath.Join(cfg.DataDir, dbFile)), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return err
	}

	// ——— Application stores (shared by router and task runner) ———
	marketStore, err := marketdata.NewStore(db)
	if err != nil {
		return fmt.Errorf("market data store: %w", err)
	}
	finStore, err := accounting.NewStore(db, router.NewInstrumentGetter(marketStore))
	if err != nil {
		return fmt.Errorf("accounting store: %w", err)
	}

	// ——— Task runner and cron scheduler (started inside GroupRunner task) ———
	taskRunner, err := taskrunner.NewRunner(taskrunner.Cfg{
		Parallelism: 6,
		QueueSize:   20,
		Logger:      l,
		DB:          db,
		LogDir:      filepath.Join(cfg.DataDir, "tasklogs"),
		LogLevel:    GetLogLevel(cfg.Env.LogLevel),
	})
	if err != nil {
		return fmt.Errorf("task runner: %w", err)
	}

	backupDest := filepath.Join(cfg.DataDir, backupsDir)
	scheduleStore, err := taskrunner.NewScheduleStore(db)
	if err != nil {
		return fmt.Errorf("schedule store: %w", err)
	}
	registerFuncs := buildTaskRegisterFuncs(taskRunner, finStore, marketStore, backupDest, l, cfg.Env.Production)
	enqueuer := taskrunner.FuncEnqueuer(func(_ context.Context, taskName string) error {
		fn := registerFuncs[taskName]
		if fn == nil {
			return fmt.Errorf("unknown task: %s", taskName)
		}
		_, err := fn()
		return err
	})
	scheduler, err := taskrunner.NewScheduler(taskrunner.SchedulerCfg{
		ScheduleStore: scheduleStore,
		Enqueuer:      enqueuer,
		Logger:        l,
	})
	if err != nil {
		return fmt.Errorf("task scheduler: %w", err)
	}

	// ——— Auth: user store, session store, session manager ———
	userStore, err := getUserStore(cfg, l)
	if err != nil {
		return err
	}
	store, _ := sessionauth.NewFsStore(filepath.Join(cfg.DataDir, sessionsDir), cfg.Auth.HashKeyBytes, cfg.Auth.BlockKeyBytes)
	sessionAuth, _ := sessionauth.New(sessionauth.Cfg{
		Store:         store,
		SessionDur:    time.Hour,       // time the user is logged in
		MaxSessionDur: 24 * time.Hour,  // time after the user is forced to re-login anyway
		MinWriteSpace: 2 * time.Minute, // throttle write operations on the session
	})

	// ——— Router (API, SPA, handlers) ———
	routerCfg := router.Cfg{
		Db:          db,
		SessionAuth: sessionAuth,
		UserMngr: userauth.LoginHandler{
			UserStore: userStore,
		},
		Logger:            l,
		BackupDestination: backupDest,
		ProductionMode:    cfg.Env.Production,
		AppSettings: handlers.AppSettings{
			DateFormat:   cfg.Settings.DateFormat,
			MainCurrency: cfg.Settings.MainCurrency,
			Currencies:   cfg.Settings.Currencies,
			Instruments:  cfg.Settings.Instruments,
		},
		TaskRunner:    taskRunner,
		ScheduleStore: scheduleStore,
		Scheduler:     scheduler,
		Enqueuers:     registerFuncs,
		TaskLogGetter: taskrunner.NewFileTaskLogReader(filepath.Join(cfg.DataDir, "tasklogs")),
		FinStore:      finStore,
		MarketStore:   marketStore,
	}
	mainAppHandler, err := router.New(routerCfg)
	if err != nil {
		return fmt.Errorf("unable to create initialize main app handler:%v", err)
	}

	// ——— GroupRunner: main server, observability server, task runner ———
	mainSrv := &http.Server{
		Addr:              cfg.Server.Addr(),
		Handler:           mainAppHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}
	obsSrv := &http.Server{
		Addr:              cfg.Obs.Addr(),
		Handler:           handlers.Admin(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	rg := tempo.NewGroupRunner()
	rg.Add(tempo.TaskDef{
		Name: "main-http",
		Run:  runHTTPServerUntilContextDone(mainSrv, cfg.Server.Addr(), l, "server"),
	})
	rg.Add(tempo.TaskDef{
		Name: "obs-http",
		Run:  runHTTPServerUntilContextDone(obsSrv, cfg.Obs.Addr(), l, "observability"),
	})
	rg.Add(tempo.TaskDef{
		Name: "taskrunner",
		Run: func(ctx context.Context) error {
			taskRunner.Start()
			scheduler.Start(ctx)
			<-ctx.Done()
			scheduler.Stop()
			// Use a fresh context with timeout so Shutdown actually waits for running tasks
			// to finish. The GroupRunner already cancelled ctx, so passing it would make
			// tempo.ShutDown return immediately without waiting for workers.
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
			defer cancel()
			return taskRunner.Shutdown(shutdownCtx)
		},
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		if err := rg.Stop(ctx); err != nil {
			l.Warn("group runner stop", slog.String("component", "server"), slog.String("error", err.Error()))
		}
	}()

	return rg.Run()
}

// runHTTPServerUntilContextDone starts srv on addr (using net.Listen), logs, serves until ctx is done, then shuts down.
func runHTTPServerUntilContextDone(srv *http.Server, addr string, l *slog.Logger, component string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return fmt.Errorf("%s listen: %w", component, err)
		}
		serveErr := make(chan error, 1)
		go func() {
			serveErr <- srv.Serve(ln)
		}()
		l.Info(fmt.Sprintf("%s server started", component), slog.String("component", component), slog.String("addr", addr))
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			l.Warn(fmt.Sprintf("%s server shutdown", component), slog.String("component", component), slog.String("error", err.Error()))
		}
		l.Info(fmt.Sprintf("%s server stopped", component), slog.String("component", component))
		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func getUserStore(cfg AppCfg, l *slog.Logger) (userauth.UserGetter, error) {
	var userGet userauth.UserGetter
	// load the correct user manager
	switch cfg.Auth.UserStore.StoreType {
	case "static":
		staticUsers := staticusers.Users{}
		for _, u := range cfg.Auth.UserStore.Users {
			staticUsers.Add(staticusers.User{
				Id:      u.Name,
				HashPw:  userauth.MustHashPw(u.Pw),
				Enabled: true,
			})
		}

		l.Debug("loading static users", slog.String("component", "users"),
			slog.Int("amount", len(staticUsers.Users)))
		userGet = &staticUsers

	case "file":

		if cfg.Auth.UserStore.FilePath == "" {
			return userGet, fmt.Errorf("no path for users file is empty")
		}
		users, err := staticusers.FromFile(cfg.Auth.UserStore.FilePath)
		if err != nil {
			return userGet, err
		}
		userGet = users
		l.Debug("loading users from file", slog.String("component", "users"),
			slog.Int("amount", len(users.Users)),
			slog.String("file", cfg.Auth.UserStore.FilePath))
	default:
		return userGet, fmt.Errorf("wrong user store in configuration, %s is not supported", cfg.Auth.UserStore.StoreType)
	}
	return userGet, nil
}

func initDataDir(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	info, err := os.Stat(absPath)
	if os.IsNotExist(err) {
		// Create the directory (and any missing parents)
		if err := os.MkdirAll(absPath, 0750); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", absPath)
	}

	// create backup dir
	backupInfo, err := os.Stat(filepath.Join(absPath, backupsDir))
	if os.IsNotExist(err) {
		// Create the directory (and any missing parents)
		if err := os.MkdirAll(filepath.Join(absPath, backupsDir), 0750); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	} else if !backupInfo.IsDir() {
		return fmt.Errorf("backup path is not a directory: %s", absPath)
	}

	// create sessions dir
	sessionsDirInfo, err := os.Stat(filepath.Join(absPath, sessionsDir))
	if os.IsNotExist(err) {
		// Create the directory (and any missing parents)
		if err := os.MkdirAll(filepath.Join(absPath, sessionsDir), 0750); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to stat path: %w", err)
	} else if !sessionsDirInfo.IsDir() {
		return fmt.Errorf("sessions path is not a directory: %s", absPath)
	}

	return nil
}
