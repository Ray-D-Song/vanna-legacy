package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Ray-D-Song/vanna-legacy/go/internal/adapters/database/duckdb"
	"github.com/Ray-D-Song/vanna-legacy/go/internal/adapters/database/mysql"
	embedopenai "github.com/Ray-D-Song/vanna-legacy/go/internal/adapters/embedder/openai"
	llmopenai "github.com/Ray-D-Song/vanna-legacy/go/internal/adapters/llm/openai"
	"github.com/Ray-D-Song/vanna-legacy/go/internal/adapters/vector/chromem"
	"github.com/Ray-D-Song/vanna-legacy/go/internal/config"
	"github.com/Ray-D-Song/vanna-legacy/go/internal/engine"
	"github.com/Ray-D-Song/vanna-legacy/go/internal/ports"
	"github.com/Ray-D-Song/vanna-legacy/go/internal/session"
	apihttp "github.com/Ray-D-Song/vanna-legacy/go/internal/transport/http"
)

func main() {
	os.Exit(run())
}

func run() int {
	if len(os.Args) < 2 {
		printUsage()
		return 1
	}
	switch os.Args[1] {
	case "serve":
		return serve(os.Args[2:])
	default:
		printUsage()
		return 1
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: vanna serve --config config.yaml\n")
}

func serve(args []string) int {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	configPath := fs.String("config", "config.yaml", "path to config file")
	_ = fs.Parse(args)

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config", "error", err)
		return 1
	}

	llmClient := llmopenai.NewClient(cfg.LLMBaseURL(), cfg.LLMAPIKey())
	embedder := embedopenai.NewClient(cfg.EmbeddingBaseURL(), cfg.EmbeddingAPIKey(), cfg.Embedding.Model, cfg.Embedding.Dimension)

	vectorStore, err := chromem.NewStore(cfg.Vector.Path, embedder, cfg.Vector.NResults.DDL, cfg.Vector.NResults.Documentation, cfg.Vector.NResults.SQL)
	if err != nil {
		slog.Error("init vector store", "error", err)
		return 1
	}

	var runner ports.SQLRunner
	if strings.TrimSpace(cfg.Database.DSN) != "" {
		switch strings.ToLower(cfg.Database.Driver) {
		case "mysql":
			runner, err = mysql.Open(cfg.Database.DSN)
		case "duckdb":
			runner, err = duckdb.Open(cfg.Database.DSN)
		default:
			slog.Error("unsupported database driver", "driver", cfg.Database.Driver)
			return 1
		}
		if err != nil {
			slog.Error("init database", "error", err)
			return 1
		}
		defer runner.Close()
	}

	svc := engine.New(engine.Options{
		LLM:    llmClient,
		Vector: vectorStore,
		Runner: runner,
		Dialect: cfg.SQLDialect(),
		Language: cfg.Engine.Language,
		MaxTokens: cfg.LLM.MaxTokens,
		Chat: ports.ChatOptions{
			Model:       cfg.LLM.Model,
			Temperature: cfg.LLM.Temperature,
			MaxTokens:   cfg.LLM.MaxTokens,
		},
		NResults: engine.NResults{
			DDL:           cfg.Vector.NResults.DDL,
			Documentation: cfg.Vector.NResults.Documentation,
			SQL:           cfg.Vector.NResults.SQL,
		},
		AllowLLMToSeeData: cfg.Engine.AllowLLMToSeeData,
		AutoTrain:         cfg.Engine.AutoTrain,
		Visualize:         cfg.Engine.Visualize,
	})

	sessions := session.NewStore(cfg.Server.SessionTTL)
	server := apihttp.NewServer(svc, sessions)

	httpServer := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		slog.Info("starting server", "addr", cfg.Server.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("shutdown", "error", err)
		return 1
	}
	return 0
}
