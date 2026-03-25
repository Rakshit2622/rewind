package cli

import (
	"context"
	"fmt"
	"os"
	"rewind/internal/snapshot"
	"rewind/internal/watcher"

	"github.com/spf13/cobra"
)

// Execute is the entry point for the CLI. It parses os.Args and dispatches
// to the appropriate command handler.
func Execute() {
	root := &cobra.Command{
		Use:   "rewind",
		Short: "File version tracker for AI-assisted coding",
	}
	root.SilenceUsage = true
	root.SilenceErrors = true
	root.AddCommand(trackCmd(), saveCmd(), historyCmd(), revertCmd(), diffCmd(), watchCmd())
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

// runTrack handles: rewind track <file>
func runTrack(args []string) error {
	_, err := snapshot.Track(args[0])
	if err != nil {
		return err
	}
	fmt.Printf("tracking %s\n", args[0])
	return nil
}

// runSave handles: rewind save <file> "<message>"
func runSave(args []string) error {
	err := snapshot.Save(args[0], args[1])
	if err != nil {
		return err
	}
	fmt.Printf("saved snapshot of %s: %q\n", args[0], args[1])
	return nil
}

// runHistory handles: rewind history <file>
func runHistory(args []string) error {
	versions, err := snapshot.History(args[0])
	if err != nil {
		return err
	}
	if len(versions) == 0 {
		fmt.Printf("no snapshots for %s\n", args[0])
		return nil
	}
	for _, v := range versions {
		fmt.Println(v)
	}
	return nil
}

// runRevert handles: rewind revert <file> <versionID>
func runRevert(args []string) error {
	err := snapshot.Revert(args[0], args[1])
	if err != nil {
		return err
	}
	fmt.Printf("reverted %s to %s\n", args[0], args[1])
	return nil
}

// runDiff handles: rewind diff <file> <version>
func runDiff(args []string) error {
	str, err := snapshot.Diff(args[0], args[1])
	if err != nil {
		return err
	}
	fmt.Println(str)
	return nil
}

func runWatch(args []string, isDiff bool) error {
	eventCh := make(chan watcher.WatchEvent, 20)

	go func() {
		for evt := range eventCh {
			switch evt.Type {
			case "saved":
				fmt.Println("✅ snapshot saved")
			case "initial_save":
				fmt.Println("✅ Initial save")
			case "skipped":
				fmt.Println("⏭ no changes")

			case "save_error":
				fmt.Println("❌ save error:", evt.Err)

			case "fs_error":
				fmt.Println("⚠️ watcher error:", evt.Err)
			case "diff_preview":
				fmt.Println("Diff preview:", evt.Data)
			}
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher.Watch(ctx, args[0], 1000, eventCh, isDiff)
	return nil
}

func diffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <file> <version>",
		Short: "Show diff between current file and a saved version",
		Args:  cobra.ExactArgs(2),
		RunE:  func(cmd *cobra.Command, args []string) error { return runDiff(args) },
	}
}

func trackCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "track <file>",
		Short: "Start tracking a file",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return runTrack(args) },
	}
}

func saveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "save <file> <message>",
		Short: "Save a snapshot of a file",
		Args:  cobra.ExactArgs(2),
		RunE:  func(cmd *cobra.Command, args []string) error { return runSave(args) },
	}
}

func revertCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "revert <file> <version>",
		Short: "Revert to a prev Version of the file",
		Args:  cobra.ExactArgs(2),
		RunE:  func(cmd *cobra.Command, args []string) error { return runRevert(args) },
	}
}

func historyCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "history <file>",
		Short: "Show version history of a file",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return runHistory(args) },
	}
}

func watchCmd() *cobra.Command {
	var isDiff bool
	cmd := &cobra.Command{
		Use:   "watch <file>",
		Short: "Watch the file for changes and make automatic snapshots",
		Args:  cobra.ExactArgs(1),
		RunE:  func(cmd *cobra.Command, args []string) error { return runWatch(args, isDiff) },
	}
	cmd.Flags().BoolVarP(&isDiff, "diff", "d", false, "Show diff previews")
	return cmd
}
