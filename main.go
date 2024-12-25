package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"

	"golang.org/x/sync/errgroup"
)

func main() {
	jobs := flag.Int("j", 10, "number of pull jobs")
	flag.Parse()
	if len(flag.Args()) != 1 {
		fmt.Println("Please provide directory")
		os.Exit(1)
	}
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	path := flag.Arg(0)
	entries, err := os.ReadDir(path)
	if err != nil {
		fmt.Println("Error reading directory: %w", err)
		os.Exit(1)
	}
	gr, grCtx := errgroup.WithContext(ctx)
	gr.SetLimit(*jobs)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		repoPath := filepath.Join(path, e.Name())
		gr.Go(func() error {
			return pull(grCtx, os.Stdout, repoPath)
		})
	}
	err = gr.Wait()
	if err != nil {
		fmt.Println("Error pulling repository: %w", err)
		os.Exit(1)
	}
}

func pull(ctx context.Context, out io.Writer, repoPath string) error {
	_, err := os.Stat(filepath.Join(repoPath, ".git"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("error checking if git repo: %w", err)
	}
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "git", "pull", "-p")
	cmd.Dir = repoPath
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("error running git pull repo: %w", err)
	}
	_, err = io.Copy(out, &buf)
	if err != nil {
		return fmt.Errorf("error printing output: %w", err)
	}
	return nil
}
