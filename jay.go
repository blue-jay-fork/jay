// Package main is the entry point for the Blue Jay command-line tool called
// Jay.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/blue-jay-fork/core/env"
	"github.com/blue-jay-fork/core/file"
	"github.com/blue-jay-fork/core/find"
	"github.com/blue-jay-fork/core/generate"
	"github.com/blue-jay-fork/core/jsonconfig"
	"github.com/blue-jay-fork/core/replace"
	"github.com/blue-jay-fork/core/storage"
	mysqlMigration "github.com/blue-jay-fork/core/storage/migration/mysql"

	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	app = kingpin.New("jay", "A command-line application to build faster with Blue Jay.")

	flagConfigFile = app.Flag("config", "Path to the env.json file.").Short('c').String()

	cFind          = app.Command("find", "Search for files containing matching text.")
	cFindFolder    = cFind.Arg("folder", "Folder to search").Required().String()
	cFindText      = cFind.Arg("text", "Case-sensitive text to find.").Required().String()
	cFindExtension = cFind.Arg("extension", "File name or extension to search in. Use * as a wildcard. Directory names are not valid.").Default("*.go").String()
	cFindRecursive = cFind.Arg("recursive", "True to search in subfolders. Default: true").Default("true").Bool()
	cFindFilename  = cFind.Arg("filename", "True to include file path in results if matched. Default: false").Default("false").Bool()

	cReplace          = app.Command("replace", "Search for files containing matching text and then replace it with new text.")
	cReplaceFolder    = cReplace.Arg("folder", "Folder to search").Required().String()
	cReplaceFind      = cReplace.Arg("find", "Case-sensitive text to replace.").Required().String()
	cReplaceText      = cReplace.Arg("replace", "Text to replace with.").String()
	cReplaceExtension = cReplace.Arg("extension", "File name or extension to search in. Use * as a wildcard. Directory names are not valid.").Default("*.go").String()
	cReplaceRecursive = cReplace.Arg("recursive", "True to search in subfolders. Default: true").Default("true").Bool()
	cReplaceFilename  = cReplace.Arg("filename", "True to include file path in results if matched. Default: false").Default("false").Bool()
	cReplaceCommit    = cReplace.Arg("commit", "True to makes the changes instead of just displaying them. Default: true").Default("true").Bool()

	cEnv          = app.Command("env", "Manage the environment config file.")
	cEnvMake      = cEnv.Command("make", "Create a new env.json file.")
	cEnvKeyshow   = cEnv.Command("keyshow", "Show a new set of session keys.")
	cEnvKeyUpdate = cEnv.Command("keyupdate", "Update env.json with a new set of session keys.")

	cMigrateMySQL         = app.Command("migrate:mysql", "Migrate MySQL to different states using 'up' and 'down' files.")
	cMigrateMySQLMake     = cMigrateMySQL.Command("make", "Create a migration file.")
	cMigrateMySQLMakeDesc = cMigrateMySQLMake.Arg("description", "Description for the migration file. Spaces will be converted to underscores and all characters will be make lowercase.").Required().String()
	cMigrateMySQLAll      = cMigrateMySQL.Command("all", "Run all 'up' files to advance the database to the latest.")
	cMigrateMySQLReset    = cMigrateMySQL.Command("reset", "Run all 'down' files to rollback the database to empty.")
	cMigrateMySQLRefresh  = cMigrateMySQL.Command("refresh", "Run all 'down' files and then 'up' files so the database is fresh and updated.")
	cMigrateMySQLStatus   = cMigrateMySQL.Command("status", "View the last 'up' file performed on the database.")
	cMigrateMySQLUp       = cMigrateMySQL.Command("up", "Apply only the next 'up' file to the database to advance the database one iteration.")
	cMigrateMySQLDown     = cMigrateMySQL.Command("down", "Apply only the current 'down' file to the database to rollback the database one iteration.")

	cGenerate     = app.Command("generate", "Generate files from template pairs.")
	cGenerateTmpl = cGenerate.Arg("folder/template", "Template pair name. Don't include an extension.").Required().String()
	cGenerateVars = stringList(cGenerate.Arg("key:value", "Key and value required for the template pair."))
)

// init sets runtime settings.
func init() {
	// Verbose logging with file name and line number
	log.SetFlags(log.Lshortfile)

	// Use all CPU cores
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	app.Version("0.5-bravo")
	app.VersionFlag.Short('v')
	app.HelpFlag.Short('h')

	argList := os.Args[1:]
	arg := kingpin.MustParse(app.Parse(argList))

	commandFind(arg)
	commandReplace(arg)
	commandEnv(arg)
	commandMigrateMySQL(arg, argList)
	commandGenerate(arg, argList)
}

func commandFind(arg string) {
	switch arg {
	case cFind.FullCommand():
		contents, err := find.Run(cFindText,
			cFindFolder,
			cFindExtension,
			cFindRecursive,
			cFindFilename)
		if err != nil {
			app.Fatalf("%v", err)
		}

		for _, line := range contents {
			fmt.Println(line)
		}
	}
}

func commandReplace(arg string) {
	switch arg {
	case cReplace.FullCommand():
		contents, err := replace.Run(cReplaceFind,
			cReplaceFolder,
			cReplaceText,
			cReplaceExtension,
			cReplaceRecursive,
			cReplaceFilename,
			cReplaceCommit)
		if err != nil {
			app.Fatalf("%v", err)
		}

		for _, line := range contents {
			fmt.Println(line)
		}
	}
}

func commandEnv(arg string) {
	switch arg {
	case cEnvMake.FullCommand():
		err := file.Copy("env.json.example", "env.json")
		if err != nil {
			app.Fatalf("%v", err)
		}
		err = env.UpdateFileKeys("env.json")
		if err != nil {
			app.Fatalf("%v", err)
		}

		p, err := filepath.Abs(".")
		if err != nil {
			app.Fatalf("%v", err)
		}
		config := filepath.Join(p, "env.json")
		if !file.Exists(config) {
			app.Fatalf("%v", err)
		}

		fmt.Println("File, env.json, created successfully with new session keys.")
		fmt.Println("Set your environment variable, JAYCONFIG, to:")
		fmt.Println(config)
	case cEnvKeyshow.FullCommand():
		fmt.Println("Paste these into your env.json file:")
		fmt.Printf(`    "AuthKey":"%v",`+"\n", env.EncodedKey(64))
		fmt.Printf(`    "EncryptKey":"%v",`+"\n", env.EncodedKey(32))
		fmt.Printf(`    "CSRFKey":"%v",`+"\n", env.EncodedKey(32))
	case cEnvKeyUpdate.FullCommand():
		err := env.UpdateFileKeys("env.json")
		if err != nil {
			app.Fatalf("%v", err)
		}
		fmt.Println("Session keys updated in env.json.")
	}
}

func commandMigrateMySQL(arg string, argList []string) {
	if argList[0] != "migrate:mysql" {
		return
	}

	var err error

	// Config struct
	info := &storage.Info{}
	
	configFile := ""

	// Check if the config file path was passed
	if len(*flagConfigFile) > 0 {
		// Load the config from the passed file
		err = jsonconfig.Load(*flagConfigFile, info)
		// Get the config file path
		configFile = *flagConfigFile
	} else {
		// Load the config from the environment variable
		err = jsonconfig.LoadFromEnv(info)
		// Get the config file path
		configFile = os.Getenv("JAYCONFIG")
	}

	if err != nil {
		app.Fatalf("%v", err)
	}

	// Perform config validation
	if len(info.MySQL.Database) == 0 {
		app.Fatalf("%v", "Database name is missing from the config file.")
	}
	
	if len(info.MySQL.Migration.Folder) == 0 {
		app.Fatalf("%v", "Migration folder is missing from the config file.")
	}
	
	// Set to the absolute path
	info.MySQL.Migration.Folder = filepath.Join(filepath.Dir(configFile), info.MySQL.Migration.Folder)

	if !file.Exists(info.MySQL.Migration.Folder) {
		app.Fatalf("%v", "Migration folder is not found on disk.")
	}

	if len(info.MySQL.Migration.Table) == 0 {
		app.Fatalf("%v", "Migration table is missing from the config file.")
	}

	// Create a new configuration
	mysqlConfig := &mysqlMigration.Configuration{
		info.MySQL,
	}

	// Create a new migration object
	mig, err := mysqlConfig.New()
	if err != nil {
		app.Fatalf("%v", err)
	}

	switch arg {
	case cMigrateMySQLMake.FullCommand():
		err = mig.Create(*cMigrateMySQLMakeDesc)
	case cMigrateMySQLAll.FullCommand():
		err = mig.UpAll()
	case cMigrateMySQLReset.FullCommand():
		err = mig.DownAll()
	case cMigrateMySQLRefresh.FullCommand():
		if mig.Position() == 0 {
			err = mig.UpAll()
		} else {
			err = mig.DownAll()
			err = mig.UpAll()
		}
	case cMigrateMySQLStatus.FullCommand():
		fmt.Println("Last migration:", mig.Status())
	case cMigrateMySQLUp.FullCommand():
		err = mig.UpOne()
	case cMigrateMySQLDown.FullCommand():
		err = mig.DownOne()
	}

	if err != nil {
		app.Fatalf("%v", err)
	} else {
		fmt.Print(mig.Output())
	}
}

func commandGenerate(arg string, args []string) {
	if args[0] != "generate" {
		return
	}

	var err error

	// Load the config
	info := &generate.Container{}

	configFile := ""

	// Check if the config file path was passed
	if len(*flagConfigFile) > 0 {
		// Load the config from the passed file
		err = jsonconfig.Load(*flagConfigFile, info)
		configFile = *flagConfigFile
	} else {
		// Load the config from the environment variable
		err = jsonconfig.LoadFromEnv(info)
		// Get the config file path
		configFile = os.Getenv("JAYCONFIG")
	}

	if err != nil {
		app.Fatalf("%v", err)
	}

	// Get the folders
	projectFolder := filepath.Dir(configFile)
	templateFolder := filepath.Join(projectFolder, info.Generation.TemplateFolder)

	// Generate the code
	err = generate.Run(args[1:], projectFolder, templateFolder)
	if err != nil {
		app.Fatalf("%v", err)
	}
}

// *****************************************************************************
// Custom Arguments
// *****************************************************************************

// StringList is a string array.
type StringList []string

// Set appends the string to the list.
func (i *StringList) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// String returns the list.
func (i *StringList) String() string {
	return strings.Join(*i, " ")
}

// IsCumulative allows more than one value to be passed.
func (i *StringList) IsCumulative() bool {
	return true
}

// stringList accepts one or more strings as arguments.
func stringList(s kingpin.Settings) (target *StringList) {
	target = new(StringList)
	s.SetValue((*StringList)(target))
	return
}
