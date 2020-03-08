package cmd

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/cobra"
)

var (
	command   string
	ignores   []string // ignoring these files and paths when watching and triggering
	recursive bool
	notify    bool
	interval  string
)

func New() (*cobra.Command, error) {
	rootCmd := cobra.Command{
		Use:     "fswatcher",
		Short:   "fswatcher watches the specified files or directories, and a changing event will trigger the user's command",
		Long:    "fswatcher watches the specified files or directories, and a changing event will trigger the user's command",
		Example: "fswatcher --cmd 'echo ***' /tmp/foo",
		Version: "0.0.1",
		Run:     run,
	}
	cobra.MinimumNArgs(1)

	rootCmd.Flags().StringVarP(&command, "command", "c", "", "the command to execute when change event notified")
	rootCmd.Flags().StringSliceVarP(&ignores, "ignore", "i", nil, "comma separated list of files and paths to ignore")
	rootCmd.Flags().BoolVarP(&recursive, "recursive", "r", true, "watch folders recursively")
	rootCmd.Flags().BoolVarP(&notify, "notify", "n", false, "enable system notify while event triggered")
	rootCmd.Flags().StringVarP(&interval, "interval", "v", "0", "the user command only executes once during an interval, "+
		"0 represents every event will trigger the execution of user's command")

	return &rootCmd, nil
}

func run(cmd *cobra.Command, args []string) {
	// get the watched filepath
	path := args[0]
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("get the absolute file path failed: %s", err)
	}

	// check whether the watching target exists or not
	fileInfo, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Fatal("the watching file or path not exist")
		}

		log.Fatalf("the watching file or path error: %s", err)
	}

	// parsing the ignored files and paths with the absolute path
	var ignoresMapping map[string]struct{}
	if len(ignores) > 0 {
		ignoresMapping := make(map[string]struct{}, len(ignores))
		for _, item := range ignores {
			// check whether the ignoring entry contains the main watching path
			ignoreItem, err := filepath.Abs(item)
			if err != nil {
				log.Fatalf("get the absolute file path failed: %s", err)
			}
			if ignoreItem == absPath {
				log.Fatal("ignoring the watching file or directory is invalid")
			}

			ignoresMapping[item] = struct{}{}
		}
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	// whether watching the subdirectories or not
	if recursive && fileInfo.IsDir() {
		fileInfos, err := ioutil.ReadDir(absPath)
		if err != nil {
			log.Fatalf("list subdirectories error: %s", err)
		}

		for _, fileInfo := range fileInfos {
			if fileInfo.IsDir() {
				absFilePath, err := filepath.Abs(fileInfo.Name())
				if err != nil {
					log.Fatalf("get subdirectory absolute path error : %s", err)
				}

				if _, ok := ignoresMapping[absFilePath]; !ok {
					err = watcher.Add(absFilePath)
					if err != nil {
						log.Fatalf("watching directory error: %s", err)
					}
				}
			}
		}
	}

	err = watcher.Add(absPath)
	if err != nil {
		log.Fatalf("watching directory error: %s", err)
	}

	<-done
}
