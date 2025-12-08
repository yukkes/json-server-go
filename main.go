package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"json-server-go/db"
	"json-server-go/server"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/cors"
)

var version = "dev"

func main() {
	// Define flags
	var port int
	var host string
	var readOnly bool
	var noCors bool
	var noPersist bool
	var delay int
	var staticDir string
	var idField string
	var watch bool
	var routes string
	var quiet bool
	var showHelp bool
	var showVersion bool

	flag.IntVar(&port, "port", 3000, "Set port")
	flag.IntVar(&port, "p", 3000, "Set port")

	flag.StringVar(&host, "host", "localhost", "Set host")
	flag.StringVar(&host, "H", "localhost", "Set host")

	flag.BoolVar(&readOnly, "read-only", false, "Allow only GET requests")
	flag.BoolVar(&readOnly, "ro", false, "Allow only GET requests")

	flag.BoolVar(&noCors, "no-cors", false, "Disable Cross-Origin Resource Sharing")
	flag.BoolVar(&noCors, "nc", false, "Disable Cross-Origin Resource Sharing")

	flag.BoolVar(&noPersist, "no-persist", false, "Disable file writes")
	flag.BoolVar(&noPersist, "np", false, "Disable file writes")

	flag.IntVar(&delay, "delay", 0, "Add delay to responses (ms)")
	flag.IntVar(&delay, "d", 0, "Add delay to responses (ms)")

	flag.StringVar(&staticDir, "static", "", "Set static files directory")
	flag.StringVar(&staticDir, "s", "", "Set static files directory")

	flag.StringVar(&idField, "id", "id", "Set database id property (e.g. _id)")
	flag.StringVar(&idField, "i", "id", "Set database id property (e.g. _id)")

	flag.BoolVar(&watch, "watch", false, "Watch file(s)")
	flag.BoolVar(&watch, "w", false, "Watch file(s)")

	flag.StringVar(&routes, "routes", "", "Path to routes file")
	flag.StringVar(&routes, "r", "", "Path to routes file")

	flag.BoolVar(&quiet, "quiet", false, "Suppress log messages from output")
	flag.BoolVar(&quiet, "q", false, "Suppress log messages from output")

	flag.BoolVar(&showHelp, "help", false, "Show help")
	flag.BoolVar(&showHelp, "h", false, "Show help")

	flag.BoolVar(&showVersion, "version", false, "Show version number")
	flag.BoolVar(&showVersion, "v", false, "Show version number")

	// Custom usage printer to match the required CLI layout
	flag.Usage = func() {
		type Option struct {
			Short   string
			Long    string
			Desc    string
			Default string
			IsBool  bool
		}

		opts := []Option{
			{Short: "-p", Long: "--port", Desc: "Set port", Default: "3000"},
			{Short: "-H", Long: "--host", Desc: "Set host", Default: "localhost"},
			{Short: "-w", Long: "--watch", Desc: "Watch file(s)", IsBool: true},
			{Short: "-r", Long: "--routes", Desc: "Path to routes file"},
			{Short: "-s", Long: "--static", Desc: "Set static files directory"},
			{Short: "-d", Long: "--delay", Desc: "Add delay to responses (ms)", Default: "0"},
			{Short: "-i", Long: "--id", Desc: "Set database id property (e.g. _id)", Default: "id"},
			{Short: "-q", Long: "--quiet", Desc: "Suppress log messages from output", IsBool: true},
			{Short: "-h", Long: "--help", Desc: "Show help", IsBool: true},
			{Short: "-v", Long: "--version", Desc: "Show version number", IsBool: true},
			{Short: "", Long: "--read-only, --ro", Desc: "Allow only GET requests", IsBool: true},
			{Short: "", Long: "--no-cors, --nc", Desc: "Disable Cross-Origin Resource Sharing", IsBool: true},
			{Short: "", Long: "--no-persist, --np", Desc: "Disable file writes", IsBool: true},
		}

		fmt.Println("json-server [options] <source>")
		fmt.Println()
		fmt.Println("Options:")
		// Calculate padding width for left column
		leftWidth := 0
		for _, o := range opts {
			left := strings.TrimSpace(o.Short)
			if left != "" && o.Long != "" {
				left = left + ", " + o.Long
			} else if left == "" {
				left = o.Long
			} else {
				left = o.Short
			}
			if len(left) > leftWidth {
				leftWidth = len(left)
			}
		}

		// Print the options with aligned columns
		for _, o := range opts {
			left := strings.TrimSpace(o.Short)
			if left != "" && o.Long != "" {
				left = left + ", " + o.Long
			} else if left == "" {
				left = o.Long
			} else {
				left = o.Short
			}

			// Compose right-hand desc with defaults and types
			desc := o.Desc
			if o.Default != "" {
				// Quote default values if they contain non-numeric characters
				defaultStr := o.Default
				numeric := true
				for _, ch := range defaultStr {
					if ch < '0' || ch > '9' {
						numeric = false
						break
					}
				}
				if !numeric {
					defaultStr = fmt.Sprintf("\"%s\"", defaultStr)
				}
				desc = fmt.Sprintf("%s [default: %s]", desc, defaultStr)
			} else if o.IsBool {
				desc = fmt.Sprintf("%s [boolean]", desc)
			}

			fmt.Printf("  %-*s  %s\n", leftWidth, left, desc)
		}

		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  json-server db.json")
		fmt.Println("  json-server db.json --quiet # Disable access logs")
		fmt.Println()
		fmt.Println("https://github.com/yukkes/json-server-go")
	}

	// Boolean flags that don't take values - keep canonical long and short forms only
	boolFlags := map[string]bool{
		"--read-only": true, "-ro": true,
		"--no-cors": true, "-nc": true,
		"--no-persist": true, "-np": true,
		"--watch": true, "-w": true,
		"--quiet": true, "-q": true,
		"--help": true, "-h": true,
		"--version": true, "-v": true,
	}

	// Find the source file from all arguments (the first non-flag argument)
	var source string
	var sourceIndex int
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		// Skip flags (starting with -)
		if len(arg) > 0 && arg[0] == '-' {
			// Skip flag value only if it's not a boolean flag
			if !boolFlags[arg] && i+1 < len(os.Args) && len(os.Args[i+1]) > 0 && os.Args[i+1][0] != '-' {
				i++ // Skip the flag value
			}
			continue
		}
		// Found the source argument
		source = arg
		sourceIndex = i
		break
	}

	// Remove source from args before parsing flags
	if sourceIndex > 0 {
		newArgs := make([]string, 0, len(os.Args)-1)
		newArgs = append(newArgs, os.Args[0])
		newArgs = append(newArgs, os.Args[1:sourceIndex]...)
		if sourceIndex+1 < len(os.Args) {
			newArgs = append(newArgs, os.Args[sourceIndex+1:]...)
		}
		os.Args = newArgs
	}

	flag.Parse()

	if showHelp {
		flag.Usage()
		return
	}

	if showVersion {
		fmt.Println(version)
		return
	}

	if source == "" {
		fmt.Fprintln(os.Stderr, "Error: Missing <source> argument")
		fmt.Fprintln(os.Stderr, "Usage: json-server [options] <source>")
		fmt.Fprintln(os.Stderr, "Example: json-server db.json")
		os.Exit(1)
	}

	fmt.Printf("Loading %s\n", source)

	database := db.New(source)
	if err := database.Load(); err != nil {
		log.Fatalf("Error loading database: %v", err)
	}

	config := &server.Config{
		Port:      port,
		Host:      host,
		ReadOnly:  readOnly,
		NoCors:    noCors,
		Delay:     delay,
		StaticDir: staticDir,
		Routes:    routes,
		IdField:   idField,
		Quiet:     quiet,
		NoPersist: noPersist,
	}

	srv := server.New(database, config)
	srv.InitRoutes()

	if watch {
		go func() {
			watcher, err := fsnotify.NewWatcher()
			if err != nil {
				log.Fatal(err)
			}
			defer watcher.Close()

			err = watcher.Add(source)
			if err != nil {
				log.Printf("Warning: Could not watch file %s: %v", source, err)
				return
			}

			fmt.Printf("Watching %s for changes...\n", source)

			for {
				select {
				case event, ok := <-watcher.Events:
					if !ok {
						return
					}
					if event.Op&fsnotify.Write == fsnotify.Write {
						log.Println("File changed, reloading...")
						// Small delay to ensure write is complete
						// time.Sleep(100 * time.Millisecond)
						if err := database.Load(); err != nil {
							log.Printf("Error reloading database: %v", err)
						} else {
							srv.InitRoutes()
							log.Println("Routes reloaded")
						}
					}
				case err, ok := <-watcher.Errors:
					if !ok {
						return
					}
					log.Println("error:", err)
				}
			}
		}()
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	fmt.Printf("JSON Server is running on http://%s\n", addr)
	fmt.Println("Resources:")
	for key := range database.GetData() {
		fmt.Printf("  http://%s/%s\n", addr, key)
	}
	if staticDir != "" {
		fmt.Printf("  Serving static files from %s\n", staticDir)
	}

	var handler http.Handler = srv
	if !noCors {
		c := cors.New(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"*"},
			AllowCredentials: true,
		})
		handler = c.Handler(srv)
	}

	log.Fatal(http.ListenAndServe(addr, handler))
}
