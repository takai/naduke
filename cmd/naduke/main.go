package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"naduke/internal/naduke"
)

func usage(fs *flag.FlagSet) func() {
	return func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [options] FILE...\n", fs.Name())
		fs.PrintDefaults()
	}
}

func parseArgs(args []string) (naduke.Options, []string, bool, *flag.FlagSet, error) {
	opts := naduke.Options{
		Host:          naduke.DefaultHost,
		Port:          naduke.DefaultPort,
		Model:         naduke.DefaultModel,
		Temperature:   naduke.DefaultTemperature,
		TopK:          naduke.DefaultTopK,
		TopP:          naduke.DefaultTopP,
		RepeatPenalty: naduke.DefaultRepeatPenalty,
		DryRun:        false,
	}

	fs := flag.NewFlagSet("naduke", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	fs.Usage = usage(fs)

	help := fs.Bool("help", false, "Show this help message and exit")
	helpShort := fs.Bool("h", false, "Show this help message and exit")

	fs.StringVar(&opts.Host, "host", opts.Host, "Ollama host (default: "+opts.Host+")")
	fs.IntVar(&opts.Port, "port", opts.Port, "Ollama port (default: "+fmt.Sprint(opts.Port)+")")
	fs.StringVar(&opts.Server, "server", "", "Full Ollama server URL (overrides host/port)")
	fs.StringVar(&opts.Model, "model", opts.Model, "Model name (default: "+opts.Model+")")
	fs.Float64Var(&opts.Temperature, "temperature", opts.Temperature, "Sampling temperature (default: 0.0)")
	fs.IntVar(&opts.TopK, "top_k", opts.TopK, "Top-k sampling (default: 1)")
	fs.Float64Var(&opts.TopP, "top_p", opts.TopP, "Top-p sampling (default: 1.0)")
	fs.Float64Var(&opts.RepeatPenalty, "repeat_penalty", opts.RepeatPenalty, "Repeat penalty (default: 1.0)")
	fs.BoolVar(&opts.DryRun, "dry-run", opts.DryRun, "Show suggested names without renaming")

	if err := fs.Parse(args); err != nil {
		return opts, nil, false, fs, err
	}

	if *help || *helpShort {
		return opts, nil, true, fs, nil
	}

	files := fs.Args()
	if len(files) == 0 {
		return opts, nil, false, fs, fmt.Errorf("no files provided")
	}

	return opts, files, false, fs, nil
}

func main() {
	opts, files, help, fs, err := parseArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Println()
		usage(fs)()
		os.Exit(1)
	}
	if help {
		usage(fs)()
		return
	}

	client, err := naduke.NewClient(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	for _, path := range files {
		if strings.TrimSpace(path) == "" {
			fmt.Fprintln(os.Stderr, "Error: empty file path")
			os.Exit(1)
		}

		sample, err := naduke.ReadSample(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		text, err := naduke.EnsureTextSample(sample, path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		rawName, err := client.GenerateName(opts.Model, opts.Temperature, opts.TopK, opts.TopP, opts.RepeatPenalty, text)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}

		newName := naduke.SanitizeName(rawName)
		destination := naduke.DestinationPath(path, newName)

		if opts.DryRun {
			fmt.Printf("%s -> %s\n", path, destination)
			continue
		}

		if err := naduke.RenameFile(path, newName); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	}
}
