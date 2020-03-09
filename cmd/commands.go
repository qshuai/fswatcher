package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/qshuai/tcolor"
	"github.com/spf13/cobra"
)

var (
	// flags
	command   string
	ignores   []string // ignoring these files and paths when watching and triggering
	recursive bool
	notify    bool
	interval  string

	changed int32
)

func New() (*cobra.Command, error) {
	rootCmd := cobra.Command{
		Use:     "fswatcher",
		Short:   "fswatcher watches the specified files or directories, and any changing event will trigger the user's command",
		Long:    "fswatcher watches the specified files or directories, and any changing event will trigger the user's command",
		Example: "fswatcher --cmd 'echo ***' /tmp/foo",
		Version: "0.0.1",
		Run:     run,
	}
	cobra.MinimumNArgs(1)

	rootCmd.Flags().StringVarP(&command, "command", "c", "", "the command to execute when change event notified")
	rootCmd.Flags().StringSliceVarP(&ignores, "ignore", "i", nil, "comma separated list of files and paths to ignore")
	rootCmd.Flags().BoolVarP(&recursive, "recursive", "r", true, "watch folders recursively")
	rootCmd.Flags().BoolVarP(&notify, "notify", "n", false, "enable system notify while event triggered")
	rootCmd.Flags().StringVarP(&interval, "interval", "v", "", "the user command only executes once during an interval, "+
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
	var ignoresMapping sync.Map
	if len(ignores) > 0 {
		for _, item := range ignores {
			// check whether the ignoring entry contains the main watching path
			ignoreItem, err := filepath.Abs(item)
			if err != nil {
				log.Fatalf("get the absolute file path failed: %s", err)
			}
			if ignoreItem == absPath {
				log.Fatal("ignoring the watching file or directory is invalid")
			}

			ignoresMapping.Store(item, nil)
		}
	}

	// user specified trigger interval
	var ticker *time.Ticker
	if interval != "" {
		duration, err := time.ParseDuration(interval)
		if err != nil {
			log.Fatalf("parsing the option<inteval> error: %s", err)
		}

		if duration.Nanoseconds() < 1 {
			log.Fatalf("less than 1 nanosecond for interval option is invalid")
		}

		ticker = time.NewTicker(duration)
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
				if recursive && event.Op&fsnotify.Create == fsnotify.Create {
					// check whether making directory or not
					stat, err := os.Stat(event.Name)
					if err != nil {
						if os.IsNotExist(err) {
							continue
						}

						log.Fatalf("get new created file error: %s", err)
					}

					if stat.IsDir() {
						// watch the new directory recursively
						err = watchRecursively(watcher, event.Name, ignoresMapping)
						if err != nil {
							log.Fatal(err)
						}
					}
				}

				if ticker != nil {
					atomic.StoreInt32(&changed, 1)
				} else {
					// execute user command
					err = execCmd(command)
					if err != nil {
						log.Fatalf("execute user command error: %s", err)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Fatalf("encountering error: %s", err)
			}
		}
	}()

	go func() {
		if ticker != nil {
			for {
				select {
				case <-ticker.C:
					if atomic.LoadInt32(&changed) == 1 {
						// execute user command
						err = execCmd(command)
						if err != nil {
							log.Fatalf("execute user command error: %s", err)
						}

						atomic.StoreInt32(&changed, 0)
					}
				}
			}
		}
	}()

	// whether watching the subdirectories or not
	if recursive && fileInfo.IsDir() {
		err = watchRecursively(watcher, absPath, ignoresMapping)
		if err != nil {
			log.Fatal(err)
		}
	}

	err = watcher.Add(absPath)
	if err != nil {
		log.Fatalf("watching directory error: %s", err)
	}
	log.Printf("watching top entry: %s", absPath)

	<-done
}

func watchRecursively(watcher *fsnotify.Watcher, absPath string, ignoresMapping sync.Map) error {
	if _, ok := ignoresMapping.Load(absPath); ok {
		return nil
	}

	fileInfos, err := ioutil.ReadDir(absPath)
	if err != nil {
		return errors.New("list subdirectories error: " + err.Error())
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			absFilePath := filepath.Join(absPath, fileInfo.Name())

			if _, ok := ignoresMapping.Load(absFilePath); !ok {
				err = watcher.Add(absFilePath)
				if err != nil {
					return errors.New("watching directory error: " + err.Error())
				}
				log.Printf("watching a subdirectory: %s", absFilePath)
			}
		}
	}

	return nil
}

func execCmd(cmd string) error {
	userCommand := exec.Command("bash", "-c", cmd)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	userCommand.Stdout = &stdout
	userCommand.Stderr = &stderr
	err := userCommand.Run()
	if err != nil {
		return err
	}

	log.Println("======== execute user command, output begin: ========")
	fmt.Printf(tcolor.WithColor(tcolor.Green, stdout.String()))
	fmt.Printf(tcolor.WithColor(tcolor.Green, stderr.String()))
	log.Println("======== execute user command, output end:   ========")

	return nil
}
