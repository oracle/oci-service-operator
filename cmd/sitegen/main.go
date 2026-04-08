package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/oracle/oci-service-operator/internal/sitegen"
)

const (
	defaultSampleRepoURL = "https://github.com/oracle/oci-service-operator"
	defaultSampleRepoRef = "main"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "sitegen: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printUsage(os.Stderr)
		return fmt.Errorf("command is required")
	}

	switch args[0] {
	case "api":
		return runAPI(args[1:])
	case "reference":
		return runReference(args[1:])
	case "verify":
		return runVerify(args[1:])
	case "help", "-h", "--help":
		printUsage(os.Stdout)
		return nil
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

type apiOptions struct {
	repoRoot      string
	outputDir     string
	sampleRepoURL string
	sampleRepoRef string
}

func runAPI(args []string) error {
	opts := apiOptions{}
	fs := flag.NewFlagSet("sitegen api", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.repoRoot, "repo-root", "", "Path to the repository root. Defaults to the detected repo root.")
	fs.StringVar(&opts.outputDir, "output-dir", "", "Directory where generated API reference Markdown is written. Defaults to <repo>/docs/reference/api.")
	fs.StringVar(&opts.sampleRepoURL, "sample-repo-url", defaultSampleRepoURL, "Repository URL used when rendering sample links.")
	fs.StringVar(&opts.sampleRepoRef, "sample-repo-ref", defaultSampleRepoRef, "Git ref used when rendering sample links.")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: sitegen api [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := resolveRepoRoot(opts.repoRoot)
	if err != nil {
		return err
	}

	if err := sitegen.GenerateAPIReferenceSite(sitegen.APIReferenceBuildOptions{
		RepoRoot:      repoRoot,
		OutputDir:     opts.outputDir,
		SampleRepoURL: opts.sampleRepoURL,
		SampleRepoRef: opts.sampleRepoRef,
	}); err != nil {
		return err
	}

	outputDir := strings.TrimSpace(opts.outputDir)
	if outputDir == "" {
		outputDir = filepath.Join(repoRoot, "docs", "reference", "api")
	}
	fmt.Fprintf(os.Stdout, "generated API reference under %s\n", outputDir)
	return nil
}

type referenceOptions struct {
	repoRoot   string
	outputRoot string
}

func runReference(args []string) error {
	opts := referenceOptions{}
	fs := flag.NewFlagSet("sitegen reference", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.repoRoot, "repo-root", "", "Path to the repository root. Defaults to the detected repo root.")
	fs.StringVar(&opts.outputRoot, "output-root", "", "Directory where generated docs/reference Markdown is written. Defaults to the repo root.")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: sitegen reference [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := resolveRepoRoot(opts.repoRoot)
	if err != nil {
		return err
	}

	result, err := sitegen.GenerateReferenceDocs(sitegen.GenerateOptions{
		Root:       repoRoot,
		OutputRoot: opts.outputRoot,
	})
	if err != nil {
		return err
	}

	outputRoot := strings.TrimSpace(opts.outputRoot)
	if outputRoot == "" {
		outputRoot = repoRoot
	}
	fmt.Fprintf(os.Stdout, "generated %d reference docs under %s\n", len(result.Written), outputRoot)
	return nil
}

type verifyOptions struct {
	repoRoot                 string
	siteDir                  string
	strictPublicDescriptions bool
}

func runVerify(args []string) error {
	opts := verifyOptions{}
	fs := flag.NewFlagSet("sitegen verify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.StringVar(&opts.repoRoot, "repo-root", "", "Path to the repository root. Defaults to the detected repo root.")
	fs.StringVar(&opts.siteDir, "site-dir", "", "Built MkDocs site directory. Defaults to <repo>/site.")
	fs.BoolVar(&opts.strictPublicDescriptions, "strict-public-descriptions", false, "Fail when customer-visible spec fields are missing descriptions.")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, "usage: sitegen verify [flags]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := resolveRepoRoot(opts.repoRoot)
	if err != nil {
		return err
	}

	result, err := sitegen.VerifyDocs(sitegen.VerifyOptions{
		RepoRoot:                 repoRoot,
		SiteDir:                  opts.siteDir,
		StrictPublicDescriptions: opts.strictPublicDescriptions,
	})
	if err != nil {
		return err
	}

	if len(result.DescriptionWarnings) > 0 {
		fmt.Fprintf(os.Stdout, "sitegen verify: %d description coverage warning(s):\n", len(result.DescriptionWarnings))
		for _, warning := range result.DescriptionWarnings {
			fmt.Fprintf(os.Stdout, "- %s\n", warning)
		}
	}

	fmt.Fprintln(os.Stdout, "docs verification passed")
	return nil
}

func resolveRepoRoot(explicitRoot string) (string, error) {
	explicitRoot = strings.TrimSpace(explicitRoot)
	if explicitRoot != "" {
		return sitegen.ResolveRepoRoot(explicitRoot)
	}
	return sitegen.ResolveRepoRoot(".")
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: sitegen <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  api    Generate docs/reference/api from checked-in CRD schemas.")
	fmt.Fprintln(w, "  reference    Generate docs/reference/ generated pages from checked-in site data.")
	fmt.Fprintln(w, "  verify    Verify generated docs drift, local/build links, and description coverage.")
}
