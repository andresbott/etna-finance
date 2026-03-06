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
	"github.com/andresbott/etna/internal/csvimport"
	"github.com/andresbott/etna/internal/marketdata"
	"github.com/andresbott/etna/internal/marketdata/importer"
	"github.com/andresbott/etna/internal/taskrunner"
	"github.com/glebarez/sqlite"
	"github.com/go-bumbu/userauth"
	"github.com/go-bumbu/userauth/handlers/sessionauth"
	"github.com/go-bumbu/userauth/userstore/staticusers"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

const dbFile = "carbon.db"
const sessionsDir = "sessions"
const backupsDir = "backup"

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
	finStore, err := accounting.NewStore(db, marketStore)
	if err != nil {
		return fmt.Errorf("accounting store: %w", err)
	}

	csvImportStore, err := csvimport.NewStore(db)
	if err != nil {
		return fmt.Errorf("csv import store: %w", err)
	}

	// ——— Instruments config vs DB consistency ———
	// If config has Instruments: false but DB contains investment/unvested accounts, override to true.
	ctx := context.Background()
	accounts, err := finStore.ListAccounts(ctx)
	if err != nil {
		return fmt.Errorf("listing accounts at startup: %w", err)
	}
	hasInvestmentAccounts := false
	for _, acc := range accounts {
		if acc.Type == accounting.InvestmentAccountType || acc.Type == accounting.UnvestedAccountType {
			hasInvestmentAccounts = true
			break
		}
	}
	if hasInvestmentAccounts && !cfg.Settings.Instruments {
		cfg.Settings.Instruments = true
		l.Warn("config discrepancy: Settings.Instruments is false but database contains investment or unvested accounts; enabling Instruments",
			slog.String("component", "startup"),
			slog.String("reason", "db_has_investment_accounts"))
	}

	// ——— Task runner and cron scheduler (started inside GroupRunner task) ———
	backupDest := filepath.Join(cfg.DataDir, backupsDir)
	taskRunner, scheduleStore, scheduler, err := initTaskRunnerAndScheduler(cfg, db, l, marketStore, finStore, backupDest)
	if err != nil {
		return err
	}

	// ——— Auth: user store, session store, session manager (or none when disabled) ———
	sessionAuth, userMngr, err := initAuth(cfg, l)
	if err != nil {
		return err
	}

	// ——— Router (API, SPA, handlers) ———
	routerCfg := router.Cfg{
		Db:                db,
		SessionAuth:       sessionAuth,
		UserMngr:          userMngr,
		AuthDisabled:      !cfg.Auth.Enabled,
		DefaultUser:       cfg.Auth.DefaultUser,
		Logger:            l,
		BackupDestination: backupDest,
		ProductionMode:    cfg.Env.Production,
		AppSettings: handlers.AppSettings{
			DateFormat:   cfg.Settings.DateFormat,
			MainCurrency: cfg.Settings.MainCurrency,
			Currencies:   cfg.Settings.AllCurrencies(),
			Instruments:  cfg.Settings.Instruments,
			Version:      metainfo.Version,
		},
		TaskRunner:    taskRunner,
		ScheduleStore: scheduleStore,
		Scheduler:     scheduler,
		TaskLogGetter: taskrunner.NewFileTaskLogReader(filepath.Join(cfg.DataDir, "tasklogs")),
		FinStore:       finStore,
		MarketStore:    marketStore,
		CsvImportStore: csvImportStore,
	}
	mainAppHandler, err := router.New(routerCfg)
	if err != nil {
		return fmt.Errorf("unable to create initialize main app handler:%v", err)
	}

	// ——— Run main server, observability server, and task runner concurrently ———
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

	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()

	g, gctx := errgroup.WithContext(rootCtx)
	g.Go(func() error { return serveHTTP(gctx, mainSrv, l, "server") })
	g.Go(func() error { return serveHTTP(gctx, obsSrv, l, "observability") })
	g.Go(func() error {
		taskRunner.Start()
		scheduler.Start(gctx)
		<-gctx.Done()
		scheduler.Stop()
		// Use a fresh context so Shutdown waits for running tasks to finish;
		// gctx is already cancelled at this point.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		return taskRunner.Shutdown(shutdownCtx)
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		rootCancel()
	}()

	return g.Wait()
}

// serveHTTP binds the listener, serves until ctx is cancelled, then shuts down gracefully.
func serveHTTP(ctx context.Context, srv *http.Server, l *slog.Logger, component string) error {
	ln, err := net.Listen("tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("%s listen: %w", component, err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()
	l.Info(component+" server started", slog.String("component", component), slog.String("addr", srv.Addr))
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		l.Warn(component+" server shutdown error", slog.String("component", component), slog.String("error", err.Error()))
	}
	l.Info(component+" server stopped", slog.String("component", component))
	if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// initTaskRunnerAndScheduler creates the task runner, registers all tasks, and starts the scheduler.
func initTaskRunnerAndScheduler(
	cfg AppCfg,
	db *gorm.DB,
	l *slog.Logger,
	marketStore *marketdata.Store,
	finStore *accounting.Store,
	backupDest string,
) (*taskrunner.Runner, *taskrunner.ScheduleStore, *taskrunner.Scheduler, error) {
	runner, err := taskrunner.NewRunner(taskrunner.Cfg{
		Parallelism: 6,
		QueueSize:   20,
		HistorySize: 20,
		Logger:      l,
		DB:          db,
		LogDir:      filepath.Join(cfg.DataDir, "tasklogs"),
		LogLevel:    GetLogLevel(cfg.Env.LogLevel),
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("task runner: %w", err)
	}

	scheduleStore, err := taskrunner.NewScheduleStore(db)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("schedule store: %w", err)
	}

	var marketDataClient importer.Client
	var fxClient importer.FXClient
	if len(cfg.MarketDataImporters.Massive.ApiKeys) > 0 {
		pool, err := importer.NewMassivePool(cfg.MarketDataImporters.Massive.ApiKeys)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("market data importer pool (massive): %w", err)
		}
		marketDataClient = pool
		fxPool, err := importer.NewMassiveFXPool(cfg.MarketDataImporters.Massive.ApiKeys)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("FX importer pool (massive): %w", err)
		}
		fxClient = fxPool
	}

	// Register tasks once; enqueue later via runner.AddRun(name) (scheduler and API).
	runner.RegisterTask(tasks.NewBackupTaskFn(finStore, backupDest, l), tasks.BackupTaskName, 0)
	runner.RegisterTask(tasks.NewFinancialImportTaskFn(marketStore, marketDataClient), tasks.FinancialImportTaskName, 0)
	runner.RegisterTask(tasks.NewFinancialBackfillTaskFn(marketStore, l, marketDataClient), tasks.FinancialBackfillTaskName, 0)
	runner.RegisterTask(tasks.NewFXImportTaskFn(marketStore, cfg.Settings.MainCurrency, cfg.Settings.AllCurrencies(), fxClient), tasks.FXImportTaskName, 0)
	runner.RegisterTask(tasks.NewFXBackfillTaskFn(marketStore, l, cfg.Settings.MainCurrency, cfg.Settings.AllCurrencies(), fxClient), tasks.FXBackfillTaskName, 0)
	if !cfg.Env.Production {
		runner.RegisterTask(tasks.NewLogOnlyTaskFn(l), tasks.LogOnlyTaskName, 4)
		runner.RegisterTask(tasks.NewLogOnlyLongTaskFn(l), tasks.LogOnlyLongTaskName, 1)
		runner.RegisterTask(tasks.NewDebugFailTaskFn(l), tasks.DebugFailTaskName, 0)
	}

	enqueuer := taskrunner.FuncEnqueuer(func(_ context.Context, taskName string) error {
		_, err := runner.AddRun(taskName)
		return err
	})
	scheduler, err := taskrunner.NewScheduler(taskrunner.SchedulerCfg{
		ScheduleStore: scheduleStore,
		Enqueuer:      enqueuer,
		Logger:        l,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("task scheduler: %w", err)
	}
	return runner, scheduleStore, scheduler, nil
}

// initAuth sets up session auth and user manager, or returns zero values when auth is disabled.
func initAuth(cfg AppCfg, l *slog.Logger) (*sessionauth.Manager, userauth.LoginHandler, error) {
	if !cfg.Auth.Enabled {
		l.Info("authentication disabled", slog.String("component", "auth"), slog.String("defaultUser", cfg.Auth.DefaultUser))
		if cfg.Env.Production {
			l.Warn("auth disabled in production mode", slog.String("component", "auth"))
		}
		return nil, userauth.LoginHandler{}, nil
	}
	userStore, err := getUserStore(cfg, l)
	if err != nil {
		return nil, userauth.LoginHandler{}, err
	}
	store, _ := sessionauth.NewFsStore(filepath.Join(cfg.DataDir, sessionsDir), cfg.Auth.HashKeyBytes, cfg.Auth.BlockKeyBytes)
	sessionMgr, _ := sessionauth.New(sessionauth.Cfg{
		Store:         store,
		SessionDur:    time.Hour,       // time the user is logged in
		MaxSessionDur: 24 * time.Hour,  // time after the user is forced to re-login anyway
		MinWriteSpace: 2 * time.Minute, // throttle write operations on the session
	})
	return sessionMgr, userauth.LoginHandler{UserStore: userStore}, nil
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
